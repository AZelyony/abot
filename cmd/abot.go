/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/cobra"
	"gopkg.in/telebot.v3"
)

const telegramMessageLimit = 4000 // Telegram message character limit

var (
	// Teletoken bot
	Teletoken = os.Getenv("TELE_TOKEN")
)

type ScanStatus struct {
	InProgress bool
	Result     string
}

var (
	statuses  = make(map[int64]*ScanStatus)
	mu        sync.Mutex
	alertFile = "alert.mp3"
)

// abotCmd represents the abot command
var abotCmd = &cobra.Command{
	Use:     "abot",
	Aliases: []string{"go"},
	Short:   "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("abot %s started \r\n", appVersion)
		abot, err := telebot.NewBot(telebot.Settings{
			URL:    "",
			Token:  Teletoken,
			Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
		})

		if err != nil {
			log.Fatalf("Please check TELE_TOKEN env variable. %s", err)
		}

		// Обработка команды /scan
		abot.Handle("/scan", func(c telebot.Context) error {
			// Разделение команды на аргументы
			args := strings.Split(c.Message().Text, " ")
			if len(args) < 2 || len(args) > 3 {
				return c.Send("Usage: /scan 192.168.0.1 [flag]")
			}

			ipRange := args[1]
			flag := ""
			if len(args) == 3 {
				flag = args[2]
			}

			// Запуск сканирования в горутине
			go func(userID int64, ipRange, flag string) {
				mu.Lock()
				statuses[userID] = &ScanStatus{InProgress: true}
				mu.Unlock()

				c.Send(fmt.Sprintf("Starting scan for range: %s with flag: %s", ipRange, flag))
				fmt.Printf("%s - starting scan for range: %s with flag: %s \r\n", time.Now().Format("2006/01/02 15:04:01"), ipRange, flag)

				// Сканирование
				result := performScan(ipRange, flag)

				mu.Lock()
				statuses[userID].InProgress = false
				statuses[userID].Result = result
				mu.Unlock()

				fmt.Printf("%s - finished scan\r\n", time.Now().Format("2006/01/02 15:04:01"))
				sendLongMessage(c, fmt.Sprintf("Scan result for %s with flag %s: %s", ipRange, flag, result))

				// Сохранение и сравнение результатов сканирования
				if saveScanResult(ipRange, flag) {
					previousScan, currentScan := getPreviousAndCurrentScans(ipRange, flag)
					if previousScan != "" && currentScan != "" {
						if scanChanged(previousScan, currentScan) {
							sendLongMessage(c, "Alert: Scan results have changed!")
							sendAlertAudio(c)
						} else {
							sendLongMessage(c, "No changes detected in the scan results.")
						}
					}
				}

			}(c.Sender().ID, ipRange, flag)

			return nil
		})

		// Обработка команды /status
		abot.Handle("/status", func(c telebot.Context) error {
			log.Print(c.Message().Payload, c.Text())
			mu.Lock()
			status, exists := statuses[c.Sender().ID]
			mu.Unlock()

			if !exists {
				return c.Send("No scan started yet.")
			}

			if status.InProgress {
				return c.Send("Scan is still in progress.")
			}

			sendLongMessage(c, fmt.Sprintf("Last scan result: %s", status.Result))
			return nil
		})

		abot.Handle(telebot.OnText, func(m telebot.Context) error {
			log.Print(m.Message().Payload, m.Text())
			//payload := m.Message().Payload
			payload := m.Text()
			var err error
			switch payload {
			case "hello", "Hello":
				err = m.Send(fmt.Sprintf("Hello I'm Abot %s!\r\n", appVersion))
			case "help", "Help":
				err = m.Send("Use command: /scan, /status")
			default:
				err = m.Send("Unknown command. Please try again.")
			}

			return err
		})

		abot.Start()
	},
}

// Функция для отправки аудиофайла
func sendAlertAudio(c telebot.Context) {
	audio := &telebot.Audio{File: telebot.FromDisk(alertFile)}
	c.Send(audio)
}

// Функция для отправки длинного сообщения
func sendLongMessage(c telebot.Context, message string) {
	for len(message) > 0 {
		if len(message) > telegramMessageLimit {
			c.Send(message[:telegramMessageLimit])
			message = message[telegramMessageLimit:]
		} else {
			c.Send(message)
			break
		}
	}
}

// Функция сканирования
func performScan(ipRange, flag string) string {
	var cmd *exec.Cmd

	// Определение команды в зависимости от флага
	switch flag {
	case "Pn":
		cmd = exec.Command("nmap", "--open", ipRange, "-Pn", "-oX", "current_scan.xml")
	case "sV":
		cmd = exec.Command("nmap", "--open", ipRange, "-sV", "-oX", "current_scan.xml")
	case "":
		cmd = exec.Command("nmap", "-sn", ipRange, "-oX", "current_scan.xml")
	default:
		return "Invalid flag. Use 'Pn', 'sV' or leave it empty."
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: %v\r\n", err)
	}
	return string(output)
}

// Сохранение результатов сканирования в файлы
func saveScanResult(ipRange, flag string) bool {
	// Замена символа "/" на "_"
	ipRange = strings.ReplaceAll(ipRange, "/", "_")

	dir := fmt.Sprintf("scans/%s/%s", ipRange, flag)
	os.MkdirAll(dir, 0755)

	files, err := os.ReadDir(dir)
	if err != nil {
		log.Println("Error reading scan directory:", err)
		return false
	}

	if len(files) >= 3 {
		os.Remove(filepath.Join(dir, files[0].Name()))
	}

	// Перемещение файла результата сканирования
	currentFile := "current_scan.xml"
	newFilename := fmt.Sprintf("%s/scan_%d.xml", dir, time.Now().Unix())
	err = os.Rename(currentFile, newFilename)
	if err != nil {
		log.Println("Error moving scan result:", err)
		return false
	}

	return true
}

// Получение двух последних сканирований
func getPreviousAndCurrentScans(ipRange, flag string) (string, string) {
	// Замена символа "/" на "_"
	ipRange = strings.ReplaceAll(ipRange, "/", "_")

	dir := fmt.Sprintf("scans/%s/%s", ipRange, flag)
	files, err := os.ReadDir(dir)
	if err != nil || len(files) < 2 {
		return "", ""
	}

	prevScan, _ := os.ReadFile(filepath.Join(dir, files[len(files)-2].Name()))
	currScan, _ := os.ReadFile(filepath.Join(dir, files[len(files)-1].Name()))
	fmt.Printf("Prev - %s \r\n", filepath.Join(dir, files[len(files)-2].Name()))
	fmt.Printf("Curr - %s \r\n", filepath.Join(dir, files[len(files)-1].Name()))

	return string(prevScan), string(currScan)
}

// Структуры для разбора XML
type NmapRun struct {
	Scanner          string   `xml:"scanner,attr"`
	Args             string   `xml:"args,attr"`
	Start            string   `xml:"start,attr"`
	StartStr         string   `xml:"startstr,attr"`
	Version          string   `xml:"version,attr"`
	XmlOutputVersion string   `xml:"xmloutputversion,attr"`
	ScanInfo         ScanInfo `xml:"scaninfo"`
	Hosts            []Host   `xml:"host"`
}

type ScanInfo struct {
	Type        string `xml:"type,attr"`
	Protocol    string `xml:"protocol,attr"`
	NumServices string `xml:"numservices,attr"`
	Services    string `xml:"services,attr"`
}

type Host struct {
	Status    Status    `xml:"status"`
	Address   Address   `xml:"address"`
	Hostnames Hostnames `xml:"hostnames"`
	Ports     Ports     `xml:"ports"`
}

type Status struct {
	State     string `xml:"state,attr"`
	Reason    string `xml:"reason,attr"`
	ReasonTTL string `xml:"reason_ttl,attr"`
}

type Address struct {
	Addr     string `xml:"addr,attr"`
	AddrType string `xml:"addrtype,attr"`
}

type Hostnames struct {
	Hostnames []Hostname `xml:"hostname"`
}

type Hostname struct {
	Name string `xml:"name,attr"`
	Type string `xml:"type,attr"`
}

type Ports struct {
	Ports []Port `xml:"port"`
}

type Port struct {
	Protocol string  `xml:"protocol,attr"`
	PortID   string  `xml:"portid,attr"`
	State    State   `xml:"state"`
	Service  Service `xml:"service"`
}

type State struct {
	State     string `xml:"state,attr"`
	Reason    string `xml:"reason,attr"`
	ReasonTTL string `xml:"reason_ttl,attr"`
}

type Service struct {
	Name    string `xml:"name,attr"`
	Product string `xml:"product,attr"`
	Version string `xml:"version,attr"`
}

// Функция для извлечения значимых данных из сканирования
func extractRelevantData(scan string) NmapRun {
	var result NmapRun
	xml.Unmarshal([]byte(scan), &result)
	return result
}

// Сравнение двух сканирований
func scanChanged(prevScan, currScan string) bool {
	prevData := extractRelevantData(prevScan)
	currData := extractRelevantData(currScan)

	// Сравнение хостов
	prevHosts := make(map[string]Host)
	currHosts := make(map[string]Host)

	for _, host := range prevData.Hosts {
		prevHosts[host.Address.Addr] = host
	}

	fmt.Printf("Prev Scan \r\n")
	fmt.Printf("%+v\n", prevHosts)

	for _, host := range currData.Hosts {
		currHosts[host.Address.Addr] = host
	}
	fmt.Printf("\r\n\r\n Curr Scan \r\n")
	fmt.Printf("%+v\n", currHosts)

	if !reflect.DeepEqual(prevHosts, currHosts) {
		// Поиск и вывод различий
		for addr, prevHost := range prevHosts {
			currHost, exists := currHosts[addr]
			if !exists {
				fmt.Printf("Host %s removed\n", addr)
			} else {
				if diff := cmp.Diff(prevHost, currHost); diff != "" {
					fmt.Printf("Differences for host %s:\n%s\n", addr, diff)
				}
			}
		}

		for addr := range currHosts {
			if _, exists := prevHosts[addr]; !exists {
				fmt.Printf("Host %s added\n", addr)
			}
		}

		return true
	}

	return false
}

func init() {
	rootCmd.AddCommand(abotCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// abotCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// abotCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
