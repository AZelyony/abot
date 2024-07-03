/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/telebot.v3"

	"github.com/hirosassa/zerodriver"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

var (
	// Teletoken bot
	Teletoken = os.Getenv("TELE_TOKEN")
	// MetricsHost exporter host:port
	MetricsHost = os.Getenv("METRICS_HOST")
)

// Initialize OpenTelemetry
func initMetrics(ctx context.Context) {

	// Create a new OTLP Metric gRPC exporter with the specified endpoint and options
	exporter, err := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithEndpoint(MetricsHost),
		otlpmetricgrpc.WithInsecure(),
	)

	if err != nil {
		// Обробка помилки, наприклад, виведення повідомлення або логування
		fmt.Printf("Failed to create exporter: %v\n", err)
		panic(err)
		// return
	}

	// Define the resource with attributes that are common to all metrics.
	// labels/tags/resources that are common to all metrics.
	resource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(fmt.Sprintf("abot_%s", appVersion)),
	)

	// Create a new MeterProvider with the specified resource and reader
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(resource),
		sdkmetric.WithReader(
			// collects and exports metric data every 10 seconds.
			sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(10*time.Second)),
		),
	)

	// Set the global MeterProvider to the newly created MeterProvider
	otel.SetMeterProvider(mp)

}

func pmetrics(ctx context.Context, payload string) {
	// Get the global MeterProvider and create a new Meter with the name "abot_light_signal_counter"
	meter := otel.GetMeterProvider().Meter("abot_light_signal_counter")

	// Get or create an Int64Counter instrument with the name "abot_light_signal_<payload>"
	counter, err := meter.Int64Counter(fmt.Sprintf("abot_light_signal_%s", payload))
	if err != nil {
		fmt.Printf("Error creating counter: %v\n", err)
		return
	}
	// Add a value of 1 to the Int64Counter
	counter.Add(ctx, 1)
}

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
		logger := zerodriver.NewProductionLogger()
		fmt.Printf("abot %s started", appVersion)

		//fmt.Println("Starting server on port 8080")
		logger.Info().Str("Version", appVersion).Msg("Starting server on port 8080")
		go func() {
			if err := http.ListenAndServe(":8080", handleRequests()); err != nil {
				//log.Fatalf("Server failed to start: %v", err)
				logger.Fatal().Str("Error", err.Error()).Msg("Server failed to start")
			}
		}()
		//fmt.Println("Started server on port 8080")
		logger.Info().Str("Version", appVersion).Msg("Started server on port 8080")

		abot, err := telebot.NewBot(telebot.Settings{
			URL:    "",
			Token:  Teletoken,
			Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
		})

		if err != nil {
			//log.Fatalf("Please check TELE_TOKEN env variable. %s", err)
			logger.Fatal().Str("Error", err.Error()).Msg("Please check TELE_TOKEN env variable.")
			return
		}

		//fmt.Println("Bot created.")
		logger.Info().Str("Version", appVersion).Msg("abot created.")

		abot.Handle("/start", func(m telebot.Context) error {
			payload := m.Message().Text
			logger.Info().Str("Command", m.Text()).Msg("Received command start")
			pmetrics(context.Background(), payload)
			return m.Send("Welcome! Type 'hello' or 'help' for more information.")
		})

		abot.Handle(telebot.OnText, func(m telebot.Context) error {
			//log.Print(m.Message().Payload, m.Text())
			logger.Info().Str("Payload", m.Text()).Msg(m.Message().Payload)
			//logger.Info().Str("Text", m.Text()).Msg("Received message")
			//payload := m.Message().Payload
			payload := m.Text()

			pmetrics(context.Background(), payload)

			var err error
			switch payload {
			case "hello", "Hello":
				err = m.Send(fmt.Sprintf("Hello I'm Abot %s!", appVersion))
			case "help", "Help":
				err = m.Send("This is the help message.")
			default:
				err = m.Send("Unknown command. Please try again.")
			}
			return err

		})

		abot.Start()
	},
}

func init() {
	ctx := context.Background()
	initMetrics(ctx)
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
