package handlers

import (
	"fmt"
	"strconv"
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

	// Get user ID to check if isAdmin = TRUE
	userID := update.Message.From.ID

	switch command {
	// --- COMMON COMMANDS ---
	case "/start":
		return "Welcome! This is KarmaBot!, ready to check your Karma :)\n\nTo get started, run /help"

	case "/help":
		return "Available commands:\n/start - Displays the Welcome message\n/help  - Displays this message\n/addkeyword - Adds keywords that the bot looks for\n/deletekeyword - Removes a keyword by its ID#\n/listkeywords - Shows the current list of word the bot looks for."

	// --- KEYWORD COMMANDS ---
	case "/addkeyword":
		// Check if sender is an admin
		if !CheckAdminRights(userID) {
			return "⛔ Permission Denied: You are not an admin."
		}
		//if admin - continue:
		if len(parts) < 2 {
			return "Usage: /addkeyword <regex pattern or word>"
		}
		pattern := strings.Join(parts[1:], " ")
		return AddKeyword(pattern, update)

	case "/deletekeyword":
		// Check if sender is an admin
		if !CheckAdminRights(userID) {
			return "⛔ Permission Denied: You are not an admin."
		}
		//if admin - continue:
		if len(parts) < 2 {
			return "Usage: /deletekeyword <id>"
		}
		return DeleteKeyword(parts[1])

	case "/listkeywords":
		// Check if sender is an admin
		if !CheckAdminRights(userID) {
			return "⛔ Permission Denied: You are not an admin."
		}
		//if admin - continue:
		return ListKeywords()

	// --- REPUTATION COMMANDS ---
	case "/checkrep":
		targetID, targetName := helperResolveTarget(update, parts)

		// If ID is -1, the target user couldnt be found via the rep map or reply-to
		score, _ := GetReputation(targetID)

		// If the name is "Unknown", try to use the name from the input
		if targetName == "" {
			targetName = "User"
		}
		return fmt.Sprintf("📊 Reputation for %s: %d", targetName, score)

	default:
		return "Sorry, I did not recognise that command, try running /help."
	}
}

// Helper: Determines who the command is targeting
// Priority: 1. Reply User, 2. @Mention or ID in args, 3. Self
func helperResolveTarget(update *tgbotapi.Update, args []string) (int64, string) {
	// 1. Check for Reply
	if update.Message.ReplyToMessage != nil {
		user := update.Message.ReplyToMessage.From
		return user.ID, user.FirstName
	}

	// 2. Check for Arguments (@User)
	// We skip the last arg if it's a number (for /setrep 100 cases)
	if len(args) > 1 {
		possibleName := args[1]
		// If arg is NOT a number (it's likely a name)
		if _, err := strconv.Atoi(possibleName); err != nil {
			id := HelperFindUserID(possibleName)
			if id != 0 {
				return id, possibleName
			}
		}
	}

	// 3. Default to Self (Sender)
	return update.Message.From.ID, update.Message.From.FirstName
}
