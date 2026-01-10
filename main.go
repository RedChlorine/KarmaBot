package main // Defines the main package for your executable application

// Import required packages for the program
import (
	"log" // Used for logging errors and information to the console
	"os"  // Used to access environment variables
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"karmabotv02/handlers"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5" // Provides the Telegram Bot API bindings for Go
	"github.com/joho/godotenv"                                    // Loads environment variables from a file
)

func main() { // Main function is the entry point of the program
	// Loads key/value pairs from .env into system environment variables
	err := godotenv.Load("Config.env")
	if err != nil {
		// If there is an error loading .env, log the error and exit the program
		log.Fatal("Error loading Config.env file")
	}

	// Retrieves the bot token from the loaded environment variables
	bot, err := tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		// If there is an error constructing the bot object (e.g., token missing or invalid), log and panic
		log.Panic(err, " -BOT TOKEN FAILED NOT FOUND OR INCORRECT")
	}

	// --- INITIALISE LOGGING --- //
	// SETUP: sets global bot pointer and loads loggin channel ID
	handlers.InitLogHandler(bot)

	// --- INITIALISE DATABASE --- //
	handlers.InitDB()

	// --- INITIALISE DB CACHE --- //
	handlers.DBInitUserCache()

	// Loads the keywords from disk from handlers/maps/mapsKeywords.json
	if err := handlers.ReloadKeywords(); err != nil {
		handlers.LogError("Warning: Could not load keywords file: %v", err)
	}

	/*
		**********************************
		        --- DEPRICATED ---
		***********************************
		// --- LOAD PIN MESSAGE DATA --- //
		//Loads pin message data from handlers/maps/mapsPinMessages.json

		handlers.LoadPinManager()
		***********************************
	*/

	// --- DEBUG MODE | Default:False --- //
	debugString := os.Getenv("DEBUG")
	isDebug, err := strconv.ParseBool(debugString)
	if err != nil {
		// If invalid or empty - default to false
		isDebug = false
	}
	bot.Debug = isDebug

	// --- STARTUP NOTIFICATION --- //
	handlers.LogInfo("🟢 KarmaBot is up and running!")

	// Creates a new Update configuration object, starting from the earliest possible update
	update := tgbotapi.NewUpdate(0)
	update.Timeout = 60 // Sets long polling timeout duration to 60 seconds

	// Requests an Updates channel based on the update configuration
	updates := bot.GetUpdatesChan(update)

	go func() {
		// Loops over incoming updates received in the channel
		for update := range updates {
			// Checks if the update contains a message (ignores other update types)
			if update.Message != nil {
				// Passive Registration
				handlers.DBEnsureUserExists(update.Message.From.ID, update.Message.From.UserName)

				//Check message for commands
				reply := handlers.CheckCommands(bot, &update)
				isCommand := reply != "" // Flag to remember if this was a command

				// 2. If not a command, Check for Keywords
				if reply == "" {
					reply = handlers.CheckMessageKeywords(&update)
				}

				if reply != "" {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, reply)

					// If it is a COMMAND in a GROUP, the user message was auto-deleted.
					// We MUST NOT reply to it, or the API will error - RACE CONDITION
					shouldReplyToMessage := true

					if isCommand && update.Message.Chat.Type != "private" {
						shouldReplyToMessage = false
					}

					// Only attach the ID if the message still exists
					if shouldReplyToMessage {
						msg.ReplyToMessageID = update.Message.MessageID
					}
					safeSend(bot, msg) // Send the reply safely - do not use bot.Send directly!
				}
			}
		}
	}()
	// --- SHUTDOWN LISTENER --- //
	// Make a channel to listen for OS signals
	sigChan := make(chan os.Signal, 1)
	// Notify this channel if we receive SIGINT (Ctrl+C) or SIGTERM (Docker stop)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Block here until a signal is received
	<-sigChan

	// 5. Cleanup & Exit
	handlers.LogInfo("🛑 Shutdown signal received. Cleaning up...")
	handlers.CloseDB() // <--- Close the DB connection!
	handlers.LogInfo("👋 Goodbye!")
}

// !!! CRITICAL HELPERS !!!  - DO NOT DELETE //
// Helper to send messages safely with 429 handling
func safeSend(bot *tgbotapi.BotAPI, msg tgbotapi.MessageConfig) {
	if msg.Text == "" {
		return
	}

	// Try to send
	_, err := bot.Send(msg)
	if err != nil {
		// Check if it is a Rate Limit Error (429)
		if err.Error() == "Too Many Requests: retry after" {
			// (Note: The actual error string might vary, usually contains "retry after")

			// Log it
			handlers.LogError("⚠️ Hit Rate Limit! Sleeping for 2 seconds...")

			// FORCE SLEEP - This saves your IP
			time.Sleep(2 * time.Second)

			// Retry once (optional)
			//bot.Send(msg)
		} else {
			handlers.LogError("❌ Error sending message: %v", err)
		}
	}
}
