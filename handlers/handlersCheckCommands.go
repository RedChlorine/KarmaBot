package handlers

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// CheckCommands looks for bot commands in the message
func CheckCommands(bot *tgbotapi.BotAPI, update *tgbotapi.Update) string {
	if update.Message == nil || update.Message.Text == "" {
		return ""
	}

	//Register the group for /pinall logic
	HelperRegisterGroup(update.Message.Chat.ID)

	// Check that command starts with "/"
	if !strings.HasPrefix(update.Message.Text, "/") {
		return ""
	}

	// Auto-delete commands in Groups and SuperGroups
	if update.Message.Chat.Type != "private" {
		deleteConfig := tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID)
		bot.Request(deleteConfig)
	}
	//---------------------------------------------------------------------------

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
		if CheckAdminRights(userID) {
			return "Available commands:\n/start - Displays the Welcome message\n/help  - Displays this message\n/top - View the Top 10 Leaderboard 🏆\n/ping - Checks if the bot is alive\n/addkeyword - Adds keywords that the bot looks for\n/addbadword - Adds  negative keywords that the bot decrements rep for\n/deletekeyword - Removes a keyword by its ID#\n/listkeywords - Shows the current list of word the bot looks for\n/checkrep - Displays the current user's reputation\n/setrep - Forces a user's rep to be set to the value you provide\n/resetrep - Resets a user's reputation to zero\n/pin - Pins a message in the group that it's replied to\n/pinall - Broadcasts the message in all groups and pins it\n/unpin - Unpins a message in the group by its ID\n/unpinall - Globally unpins all pinned messages - !CAUTION!\n"
		} else {
			return "Available commands:\n/start - Displays the Welcome message\n/help  - Displays this message\n/top - View the Top 10 Leaderboard 🏆\n/ping - Checks if the bot is alive\n"
		}

	// --- KEYWORD COMMANDS ---

	case "/ping":
		// if the bot is alive - it'll respond
		return "PONG!"

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

	case "/addbadword":
		// Check if sender is an admin
		if !CheckAdminRights(userID) {
			return "⛔ Permission Denied: You are not an admin."
		}
		//if admin - continue:
		if len(parts) < 2 {
			return "Usage: /addkeyword <regex pattern or word>"
		}
		pattern := strings.Join(parts[1:], " ")
		return AddNegativeKeyword(pattern, update)

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

	case "/top":
		// Returns up to 10 with the highest rep in the DBs
		return GetTopReputation()

	case "/pin":
		// Usage: Reply to a message and type /pin
		if !CheckAdminRights(userID) {
			return "⛔ Admin only."
		}
		if update.Message.ReplyToMessage == nil {
			return "⚠️ You must reply to the message you want to pin."
		}

		// Pass the text so it can be saved in the list
		text := update.Message.ReplyToMessage.Text
		return PinMessage(bot, update.Message.Chat.ID, update.Message.ReplyToMessage.MessageID, update.Message.From.FirstName, text)

	case "/unpin":
		// Usage: /unpin <ID>
		if !CheckAdminRights(userID) {
			return "⛔ Permission Denied: You are not an admin."
		}

		if len(parts) < 2 {
			return "Usage: /unpin <Pin ID#> (e.g. /unpin 5)"
		}

		id, err := strconv.Atoi(parts[1])
		if err != nil {
			return "Error: Pin ID must be a number."
		}
		return UnpinByID(bot, id)

	case "/unpinall":
		// Usage: /unpinall confirm
		if !CheckAdminRights(userID) {
			return "⛔ Permission Denied: You are not an admin."
		}

		// Check if the user added the "confirm" argument - early return if not passed
		if len(parts) < 2 || !strings.EqualFold(parts[1], "confirm") {
			return "⚠️ SAFETY CHECK: This will unpin ALL messages in ALL groups.\n\nTo proceed, you must type:\n`/unpinall confirm`"
		}

		// If they typed "/unpinall confirm", run the function
		return UnpinAllGlobal(bot)

	case "/pinall":
		// Usage: /pinall <message text>
		if !CheckAdminRights(userID) {
			return "⛔ Permission Denied: You are not an admin."
		}

		if len(parts) < 2 {
			return "Usage: /pinall <text to broadcast and pin>"
		}

		text := strings.Join(parts[1:], " ")
		return BroadcastAndPin(bot, text, update.Message.From.FirstName)

	case "/listpins":
		if !CheckAdminRights(userID) {
			return "⛔ Permission Denied: You are not an admin."
		}
		return HelperListPins(update.Message.Chat.ID)

	// -- DEPRICATED -- //
	/*case "/decrement":
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
	return fmt.Sprintf("🔻 Decreased %s's rep by 1. Total: %d", targetName, newReputation)*/

	default:
		return "Sorry, I did not recognise that command, try running /help."
	}
}

// --- HELPERS ---

// Helper: Determines who the command is targeting
// Priority: 1. Reply User, 2. @Mention or ID in args, 3. Self
func helperResolveTarget(update *tgbotapi.Update, args []string) (int64, string) {
	// 1. Check for Reply
	if update.Message.ReplyToMessage != nil {
		user := update.Message.ReplyToMessage.From
		return user.ID, user.UserName
	}

	// 2. Check for Arguments (@User)
	if len(args) > 1 {
		possibleName := args[1]

		// Check if arg is a Name (not a number)
		// We use Atoi to make sure we don't catch "/setrep 100" as a username
		if _, err := strconv.Atoi(possibleName); err != nil {
			id := HelperFindUserID(possibleName)
			return id, possibleName
		}
	}

	// 3. Default to Self (Sender)
	// Only reached if: No Reply AND (No args OR Arg was a number)
	return update.Message.From.ID, update.Message.From.UserName
}
