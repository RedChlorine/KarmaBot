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
		return "Available commands:\n/start - Displays the Welcome message\n/help  - Displays this message\n/addkeyword - Adds keywords that the bot looks for\n/deletekeyword - Removes a keyword by its ID#\n/listkeywords - Shows the current list of word the bot looks for\n/checkrep - Displays the current user's reputation\n/setrep - Forces a user's rep to be set to the value you provide\n/resetrep - Resets a user's reputation to zero\n/decrement - Reduces a user's reputation by one"

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
		// /checkrep [optional: @username]
		// Returns the user's current reputation
		targetID, targetName := helperResolveTarget(update, parts)

		// If is -1, the target user couldnt be found via the rep map or reply-to
		score, _ := GetReputation(targetID)

		// If the name is "Unknown", try to use the name from the input
		if targetName == "" {
			targetName = "User"
		}
		return fmt.Sprintf("📊 Reputation for %s: %d", targetName, score)

	case "/setrep":
		// /setrep <amount> (Reply or @Username)
		// Foreces a user's reputation to be a set value
		if !CheckAdminRights(userID) {
			return "⛔ Permission Denied.  You are not an admin."
		}

		if len(parts) < 2 {
			return "Usage: /setrep <amount> (reply to user)"
		}

		// Get target info
		targetID, targetName := helperResolveTarget(update, parts)
		//log.Printf("\n\nHANDLERS - COMMANDS - SETREP\nTARGRET_ID:%d\nTARGET_NAME:%s", targetID, targetName)

		if targetID == 0 {
			return fmt.Sprintf("❌ Error: User '%s' not found in Reputation map. The bot can only manage users who have spoken before.", targetName)
		}

		//parse the <amount>
		amountString := parts[len(parts)-1]
		amount, err := strconv.Atoi(amountString)
		if err != nil {
			return "Error: No user specified (Reply or @User)."
		}

		// Set new Rep
		newReputation, _ := SetReputation(targetID, targetName, amount)
		return fmt.Sprintf("✅ Set %s's reputation to %d.", targetName, newReputation)

	case "/resetrep":
		// Forces the user's rep to zero
		// Check if sender is an admin
		if !CheckAdminRights(userID) {
			return "⛔ Permission Denied: You are not an admin."
		}

		// Get target info
		targetID, targetName := helperResolveTarget(update, parts)
		if targetID == 0 {
			return "Error: No user specified."
		}

		SetReputation(targetID, targetName, 0)
		newReputation, _ := GetReputation(targetID)
		return fmt.Sprintf("🔄 Reset %s's reputation.\nReputation: %d", targetName, newReputation)

	case "/decrement":
		// Decrements a user's rep by 1
		// Check if sender is an admin
		if !CheckAdminRights(userID) {
			return "⛔ Permission Denied: You are not an admin."
		}

		// Get target info
		targetID, targetName := helperResolveTarget(update, parts)
		if targetID == 0 {
			return "Error: No user specified."
		}

		newReputation, _ := DecreaseReputation(targetID, targetName)
		return fmt.Sprintf("🔻 Decreased %s's rep by 1. Total: %d", targetName, newReputation)

	default:
		return "Sorry, I did not recognise that command, try running /help."
	}
}

// --- HELPERS ---

// Helper: Determines who the command is targeting
// Priority: 1. Reply User, 2. @Mention or ID in args, 3. Self
func helperResolveTarget(update *tgbotapi.Update, args []string) (int64, string) {
	// 1. Check for Reply
	//if update.Message.ReplyToMessage != nil {
	//	user := update.Message.ReplyToMessage.From
	//	/*******************DEBUG INFO**********************/
	//	log.Printf("User used a reply\nUSER:%d\nFIRSTNAME:%s", user.ID, user.UserName)
	//	return user.ID, user.UserName
	//}

	// 2. Check for Arguments (@User)
	if len(args) > 1 {
		possibleName := args[1]
		/*******************DEBUG INFO**********************/
		//log.Printf("\n\nHELPER - RESOLVE_TARGET\nFound possible Name: %s", possibleName)

		// Check if arg is a Name (not a number)
		// We use Atoi to make sure we don't catch "/setrep 100" as a username
		if _, err := strconv.Atoi(possibleName); err != nil {

			id := HelperFindUserID(possibleName)

			//log.Printf("\n\nHELPER - RESOLVE_TARGET\nID of possible name: %d\nUSERNAME of possible name: %s", id, possibleName)

			return id, possibleName
		}
	}

	// 3. Default to Self (Sender)
	// Only reached if: No Reply AND (No args OR Arg was a number)
	return update.Message.From.ID, update.Message.From.UserName
}
