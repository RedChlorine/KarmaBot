package handlers

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// CheckCommands looks for bot commands in the message
func CheckCommands(update *tgbotapi.Update) string {
	if update.Message == nil || update.Message.Text == "" {
		return ""
	}

	// Check that command starts with "/"
	if !strings.HasPrefix(update.Message.Text, "/") {
		return ""
	}

	// Extract the command
	parts := strings.Fields(update.Message.Text)
	command := parts[0]

	// Remove possible @(botname) suffix
	command = strings.SplitN(command, "@", 2)[0]

	switch command {
	case "/start":
		return "Welcome! This is KarmaBot!, ready to check your Karma :)"

	case "/help":
		return "Available commands:\n/start - Displays the Welcome message\n/help  - Displays this message\n/addkeyword - Adds keywords that the bot looks for\n/deletekeyword - Removes a keyword by its ID#\n/listkeywords - Shows the current list of word the bot looks for."

	case "/addkeyword":
		if len(parts) < 2 {
			return "Usage: /addkeyword <regex pattern or word>"
		}
		pattern := strings.Join(parts[1:], " ")
		return AddKeyword(pattern, update)

	case "/deletekeyword":
		if len(parts) < 2 {
			return "Usage: /deletekeyword <id>"
		}
		return DeleteKeyword(parts[1])

	case "/listkeywords":
		return ListKeywords()

	default:
		return "Sorry, I did not recognise that command, try running /help."
	}
}
