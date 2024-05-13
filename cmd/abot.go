/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/telebot.v3"
)

var (
	// Teletoken bot
	Teletoken = os.Getenv("TELE_TOKEN")
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

		fmt.Println("Starting server on port 8080")
		go func() {
			if err := http.ListenAndServe(":8080", handleRequests()); err != nil {
				log.Fatalf("Server failed to start: %v", err)
			}
		}()
		fmt.Println("Started server on port 8080")

		fmt.Printf("abot %s started\n", appVersion)

		abot, err := telebot.NewBot(telebot.Settings{
			URL:    "",
			Token:  Teletoken,
			Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
		})

		fmt.Println("Bot created.")

		if err != nil {
			log.Fatalf("Please check TELE_TOKEN env variable. %s", err)
			return
		}

		abot.Handle(telebot.OnText, func(m telebot.Context) error {
			log.Print(m.Message().Payload, m.Text())
			payload := m.Message().Payload

			switch payload {
			case "hello":
				err = m.Send(fmt.Sprintf("Hello I'm Abot %s!", appVersion))
			}

			return err
		})

		abot.Start()
	},
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

func handleRequests() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/liveness":
			// Возвращаем текущее время в формате строки
			currentTime := time.Now().Format(time.RFC3339)
			// Пишем текущее время в тело ответа
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(currentTime))
		case "/readyness":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	})
}
