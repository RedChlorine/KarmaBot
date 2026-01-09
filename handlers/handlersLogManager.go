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
			log.Printf("[INFO]✅ Log Handler Initialized! Sending logs to Channel ID: %d", logChannelID)
		}
	} else {
		log.Println("ℹ️ NOTE: LOG_CHANNEL_ID is not set in .env. Logs will only print to console.")
	}
}

// Sends an error messahe to the log channel and prints to the console
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
