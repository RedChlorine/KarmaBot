package main // Defines the main package for your executable application

// Import required packages for the program
import (
	"log" // Used for logging errors and information to the console
	"os"  // Used to access environment variables
	"strconv"

	"karmabotv02/handlers"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5" // Provides the Telegram Bot API bindings for Go
	"github.com/joho/godotenv"                                    // Loads environment variables from a file
)

func main() { // Main function is the entry point of the program
	// Loads key/value pairs from Env.env into system environment variables
	err := godotenv.Load("Env.env")
	if err != nil {
		// If there is an error loading Env.env, log the error and exit the program
		log.Fatal("Error loading Env.env file")
	}

	// Retrieves the bot token from the loaded environment variables
	bot, err := tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		// If there is an error constructing the bot object (e.g., token missing or invalid), log and panic
		log.Panic(err, " -BOT TOKEN FAILED NOT FOUND OR INCORRECT")
	}

	// Loads the keywords from disk from handlers/maps/mapsKeywords.json
	if err := handlers.LoadKeywordFromFile(); err != nil {
		log.Println("Warning: Could not load keywords file - keyword system may not function as intended:", err)
	}

	// Loads the reputation from disk from handlers/maps/mapsReputation.json
	if err := handlers.LoadReputationFromFile(); err != nil {
		log.Println("Warning: Could not load reputation file - Rep system may not function as intended:", err)
	}

	// Loads pin message data from handlers/maps/mapsPinMessages.json
	handlers.LoadPinManager()

	// --- DEBUG --- //
	bot.Debug = true
	log.Println("Bot is running and ready!") //console log if bot is running

	// --- STARTUP NOTIFICATION --- //
	logChannelID := os.Getenv("LOG_CHANNEL_ID")
	logGroupID := os.Getenv("LOG_GROUP_ID")

	if logChannelID != "" {
		logChannelID, err := strconv.Atoi(logChannelID)
		if err != nil {
			log.Println("⚠️ WARNING: Could not parse LOG_CHANNEL_ID:", err)
		} else {
			messageIsAlive := "🟢 KarmaBot is up and running!"
			logIsAlive := tgbotapi.NewMessage(int64(logChannelID), messageIsAlive)
			bot.Send(logIsAlive)
		}
	} else {
		log.Println("ℹ️ Note: LOG_CHANNEL_ID not set in Env.env, skipping startup message.")
	}

	if logGroupID != "" {
		logGroupID, err := strconv.Atoi(logGroupID)
		if err != nil {
			log.Println("⚠️ WARNING: Could not parse LOG_GROUP_ID:", err)
		} else {
			messageIsAlive := "🟢 KarmaBot is up and running!"
			logIsAlive := tgbotapi.NewMessage(int64(logGroupID), messageIsAlive)
			bot.Send(logIsAlive)
		}
	} else {
		log.Println("ℹ️ Note: LOG_GROUP_ID not set in Env.env, skipping startup message.")
		messageIsAlive := "🟢 KarmaBot is up and running! - ℹ️ BUT .ENV CONFIG FAILED TO PARSE"
		logIsAlive := tgbotapi.NewMessage(-1003600147866, messageIsAlive)
		bot.Send(logIsAlive)
	}

	// Creates a new Update configuration object, starting from the earliest possible update
	update := tgbotapi.NewUpdate(0)
	update.Timeout = 60 // Sets long polling timeout duration to 60 seconds

	// Requests an Updates channel based on the update configuration
	updates := bot.GetUpdatesChan(update)

	// Loops over incoming updates received in the channel
	for update := range updates {
		// Checks if the update contains a message (ignores other update types)
		if update.Message != nil {
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
				// ------------------------------

				bot.Send(msg)
			}
		}
	}
}
