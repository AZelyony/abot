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
	statuses = make(map[int64]*ScanStatus)
	mu       sync.Mutex
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
		fmt.Printf("abot %s started ", appVersion)
		abot, err := telebot.NewBot(telebot.Settings{
			URL:    "",
			Token:  Teletoken,
			Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
		})

		if err != nil {
			log.Fatalf("Please check TELE_TOKEN env variable. %s", err)
		}

		//		abot.Handle("/start", func(m telebot.Context) error {
		//			payload := m.Message().Text
		//			logger.Info().Str("Command", m.Text()).Msg("Received command start")

		//			return m.Send("Welcome! Type 'hello' or 'help' for more information.")
		//		})

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
				fmt.Printf("Starting scan for range: %s with flag: %s", ipRange, flag)

				// Сканирование
				result := performScan(ipRange, flag)

				mu.Lock()
				statuses[userID].InProgress = false
				statuses[userID].Result = result
				mu.Unlock()

				sendLongMessage(c, fmt.Sprintf("Scan result for %s with flag %s: %s", ipRange, flag, result))

				// Сохранение и сравнение результатов сканирования
				if saveScanResult(ipRange, flag, result) {
					previousScan, currentScan := getPreviousAndCurrentScans(ipRange, flag)
					if previousScan != "" && currentScan != "" {
						if scanChanged(previousScan, currentScan) {
							sendLongMessage(c, "Alert: Scan results have changed!")
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
				err = m.Send(fmt.Sprintf("Hello I'm Abot %s!", appVersion))
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
func performScan(ipRange string, flag string) string {

	var cmd *exec.Cmd

	// Определение команды в зависимости от флага
	switch flag {
	case "Pn":
		cmd = exec.Command("nmap", "--open", ipRange, "-Pn")
	case "sV":
		cmd = exec.Command("nmap", "--open", ipRange, "-sV")
	case "":
		cmd = exec.Command("nmap", "-sn", ipRange)
	default:
		return "Invalid flag. Use 'Pn' or 'Sv'."
	}
	output, err := cmd.Output()
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return string(output)
}

// Сохранение результатов сканирования в файлы
func saveScanResult(ipRange, flag, result string) bool {
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

	filename := fmt.Sprintf("%s/scan_%d.xml", dir, time.Now().Unix())
	err = os.WriteFile(filename, []byte(result), 0644)
	if err != nil {
		log.Println("Error writing scan result:", err)
		return false
	}

	return true
}

// Получение двух последних сканирований
func getPreviousAndCurrentScans(ipRange, flag string) (string, string) {
	dir := fmt.Sprintf("scans/%s/%s", ipRange, flag)
	files, err := os.ReadDir(dir)
	if err != nil || len(files) < 2 {
		return "", ""
	}

	prevScan, _ := os.ReadFile(filepath.Join(dir, files[len(files)-2].Name()))
	currScan, _ := os.ReadFile(filepath.Join(dir, files[len(files)-1].Name()))

	return string(prevScan), string(currScan)
}

// Структуры для разбора XML
type NmapRun struct {
	XMLName xml.Name `xml:"nmaprun"`
	Hosts   []Host   `xml:"host"`
}

type Host struct {
	XMLName   xml.Name   `xml:"host"`
	Address   Address    `xml:"address"`
	Ports     Ports      `xml:"ports"`
	Hostnames []Hostname `xml:"hostnames>hostname"`
}

type Address struct {
	XMLName xml.Name `xml:"address"`
	Addr    string   `xml:"addr,attr"`
}

type Ports struct {
	XMLName xml.Name `xml:"ports"`
	Ports   []Port   `xml:"port"`
}

type Port struct {
	XMLName xml.Name `xml:"port"`
	PortID  string   `xml:"portid,attr"`
	State   State    `xml:"state"`
	Service Service  `xml:"service"`
}

type State struct {
	XMLName xml.Name `xml:"state"`
	State   string   `xml:"state,attr"`
}

type Service struct {
	XMLName xml.Name `xml:"service"`
	Name    string   `xml:"name,attr"`
}

type Hostname struct {
	Name string `xml:"name,attr"`
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

	for _, host := range currData.Hosts {
		currHosts[host.Address.Addr] = host
	}

	return !reflect.DeepEqual(prevHosts, currHosts)
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
