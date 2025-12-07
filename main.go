package main // Defines the main package for your executable application

// Import required packages for the program
import (
	"log" // Used for logging errors and information to the console
	"os"  // Used to access environment variables

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

	// Enables verbose debug output to the console
	bot.Debug = true
	log.Println("Bot is running and ready!") //console log if bot is running

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
			reply := handlers.CheckCommands(&update)
			if reply == "" {
				// If !=command then chekc for keywords
				reply = handlers.CheckMessageKeywords(&update)
			}

			if reply != "" {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, reply)
				bot.Send(msg)
			}
		}
	}
}
