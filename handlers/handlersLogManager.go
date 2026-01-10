package handlers

import (
	"fmt"
	"log"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	// Global bot instance for handlers package
	botInstance *tgbotapi.BotAPI

	// ID of the channel to send logs to
	logChannelID int64

	// "Mailbox" (channel) for log messages - holds 100 pending logs
	logQueue = make(chan string, 100)
)

// Initialises the logger with the bot instance and loads the channel ID
func InitLogHandler(bot *tgbotapi.BotAPI) {
	botInstance = bot
	envChannelID := os.Getenv("LOG_CHANNEL_ID")

	if envChannelID != "" {
		var err error
		logChannelID, err = strconv.ParseInt(envChannelID, 10, 64)

		if err != nil {
			log.Printf("⚠️ WARNING: Could not parse LOG_CHANNEL_ID from .env: %v", err)
		} else {
			log.Printf("[INFO] ✅ Log Handler Initialized! Sending logs to Channel ID: %d", logChannelID)

			// !!- GOROUTINE -!! //
			// process log messages if channel ID is valid
			go startLogWorker()
		}
	} else {
		log.Println("ℹ️ NOTE: LOG_CHANNEL_ID is not set in .env. Logs will only print to console.")
	}
}

// Goroutine worker to process log messages from the queue - Background processing
func startLogWorker() {
	for text := range logQueue {
		if botInstance != nil && logChannelID != 0 {
			message := tgbotapi.NewMessage(logChannelID, text)
			botInstance.Send(message)
		}
	}
}

// Gouroutine-safe helper to send log messages asynchronously
func helperSendToLogChannelAsync(text string) {
	// Non-Blocking send to log queue
	select {
	case logQueue <- text:
		// Successfully sent to log queue
	default:
		// Log queue is full, drop the log message to avoid blocking
		log.Println("⚠️ WARNING: Log Queue FULL!! Dropping log message to console only:", text)
	}
}

// Sends an error message to the log channel and prints to the console
// Usage: handlers.LogError("Something went wrong: %v", err)
func LogError(format string, args ...any) {
	message := fmt.Sprintf(format, args...)

	// IMPORTANT: Always log to console //
	log.Printf("[ERROR] %s", message)

	// Send Error to log channel
	helperSendToLogChannel("⚠️ ERROR:\n\n" + message)
}

// LogInfo sends an INFO message to the log channel
func LogInfo(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	log.Printf("[INFO] %s", message)
	helperSendToLogChannel("ℹ️ INFO:\n\n" + message)
}

// Helper to send the actual message
func helperSendToLogChannel(text string) {
	if botInstance == nil || logChannelID == 0 {
		log.Println("⚠️WARNING: LOGGER INIT FAILED - NO LOGS WILL BE SENT TO LOG CHANNEL")
		return
	}

	message := tgbotapi.NewMessage(logChannelID, text)

	if _, err := botInstance.Send(message); err != nil {
		log.Printf("⚠️ Failed to send log to Telegram: %v", err)
	}
}
