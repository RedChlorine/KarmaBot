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

	// 1. Register Group in DB (for Broadcasts)
	if update.Message.Chat.Type != "private" {
		DBRegisterGroup(update.Message.Chat.ID)
	}

	// 2. Check for Prefix
	if !strings.HasPrefix(update.Message.Text, "/") {
		return ""
	}

	// 3. Auto-delete command message in groups
	if update.Message.Chat.Type != "private" {
		bot.Request(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID))
	}

	// 4. Parse Command
	parts := strings.Fields(update.Message.Text)
	command := strings.SplitN(parts[0], "@", 2)[0] // Handle /cmd@BotName
	userID := update.Message.From.ID

	switch command {
	// --- BASICS ---
	case "/start":
		return "Welcome to KarmaBot! 🤖"
	case "/help":
		if CheckAdminRights(userID) {
			return "Available commands:\n/start - Displays the Welcome message\n/help  - Displays this message\n/top - View the Top 10 Leaderboard 🏆\n/ping - Checks if the bot is alive\n/addkeyword - Adds keywords that the bot looks for\n/addbadword - Adds  negative keywords that the bot decrements rep for\n/deletekeyword - Removes a keyword by its ID#\n/listkeywords - Shows the current list of word the bot looks for\n/listpins - Lets you check the list of current pins and their IDs\n/checkrep - Displays the current user's reputation\n/setrep - Forces a user's rep to be set to the value you provide\n/resetrep - Resets a user's reputation to zero\n/pin - Pins a message in the group that it's replied to\n/pinall - Broadcasts the message in all groups and pins it\n/unpin - Unpins a message in the group by its ID\n/unpinall - Globally unpins all pinned messages - !CAUTION!\n/setupDB - Sets up the database tables - !!CAUTION!!\n"
		} else {
			return "Available commands:\n/start - Displays the Welcome message\n/help  - Displays this message\n/top - View the Top 10 Leaderboard 🏆\n/ping - Checks if the bot is alive\n"
		}

	case "/ping":
		return "PONG! 🏓"

	// --- KEYWORDS (DB) ---
	case "/addkeyword", "/addbadword":
		if !CheckAdminRights(userID) {
			return "⛔ Admin only."
		}
		if len(parts) < 2 {
			return "Usage: /addkeyword <word>"
		}

		isNeg := (command == "/addbadword")
		pattern := strings.Join(parts[1:], " ")

		id, err := DBAddKeyword(pattern, isNeg, update.Message.From.UserName)
		if err != nil {
			return fmt.Sprintf("❌ DB Error: %v", err)
		}

		ReloadKeywords() // Update Cache!
		return fmt.Sprintf("✅ Added Keyword #%d: '%s'", id, pattern)

	case "/deletekeyword":
		if !CheckAdminRights(userID) {
			return "⛔ Admin only."
		}
		if len(parts) < 2 {
			return "Usage: /deletekeyword <ID>"
		}

		id, _ := strconv.Atoi(parts[1])
		if err := DBDeleteKeyword(id); err != nil {
			return fmt.Sprintf("❌ Error: %v", err)
		}

		ReloadKeywords() // Update Cache!
		return fmt.Sprintf("🗑️ Deleted Keyword #%d", id)

	case "/listkeywords":
		if !CheckAdminRights(userID) {
			return "⛔ Admin only."
		}
		return DBListKeywords()

	// --- REPUTATION (DB) ---
	case "/checkrep":
		targetID, targetName := helperResolveTarget(update, parts)
		score, _ := DBGetReputationScore(targetID)
		if targetName == "" {
			targetName = "User"
		}
		return fmt.Sprintf("📊 Reputation for %s: %d", targetName, score)

	case "/setrep":
		if !CheckAdminRights(userID) {
			return "⛔ Admin only."
		}
		if len(parts) < 2 {
			return "Usage: /setrep <amount>"
		}

		targetID, targetName := helperResolveTarget(update, parts)
		amount, _ := strconv.Atoi(parts[len(parts)-1])

		newScore, err := DBSetReputation(targetID, targetName, amount)
		if err != nil {
			return "❌ DB Error"
		}
		return fmt.Sprintf("✅ Set %s to %d", targetName, newScore)

	case "/resetrep":
		if !CheckAdminRights(userID) {
			return "⛔ Admin only."
		}
		targetID, targetName := helperResolveTarget(update, parts)
		DBSetReputation(targetID, targetName, 0)
		return fmt.Sprintf("🔄 Reset %s to 0", targetName)

	case "/top":
		return DBGetTop10()

	// --- PINS (DB + API) ---
	case "/pin":
		if !CheckAdminRights(userID) {
			return "⛔ Admin only."
		}
		if update.Message.ReplyToMessage == nil {
			return "⚠️ Reply to a message to pin it."
		}

		reply := update.Message.ReplyToMessage
		// 1. API Call
		if _, err := bot.Request(tgbotapi.PinChatMessageConfig{ChatID: update.Message.Chat.ID, MessageID: reply.MessageID}); err != nil {
			return "⚠️ Failed to pin (Check Bot Permissions)."
		}
		// 2. DB Save
		id, _ := DBPinMessage(update.Message.Chat.ID, reply.MessageID, update.Message.From.FirstName, reply.Text)
		return fmt.Sprintf("📌 Pinned! (ID: #%d)", id)

	case "/unpin":
		if !CheckAdminRights(userID) {
			return "⛔ Admin only."
		}
		if len(parts) < 2 {
			return "Usage: /unpin <ID>"
		}

		id, _ := strconv.Atoi(parts[1])
		chatID, msgID, err := DBUnpinByID(id)
		if err != nil {
			return "⚠️ Pin ID not found."
		}

		// API Call
		bot.Request(tgbotapi.UnpinChatMessageConfig{ChatID: chatID, MessageID: msgID})
		return fmt.Sprintf("🗑️ Unpinned #%d", id)

	case "/listpins":
		return DBListPins(update.Message.Chat.ID)

	case "/pinall":
		if !CheckAdminRights(userID) {
			return "⛔ Admin only."
		}
		if len(parts) < 2 {
			return "Usage: /pinall <text>"
		}
		text := strings.Join(parts[1:], " ")
		return helperBroadcastAndPin(bot, text, update.Message.From.FirstName)

	case "/unpinall":
		if !CheckAdminRights(userID) {
			return "⛔ Admin only."
		}
		if len(parts) < 2 || parts[1] != "confirm" {
			return "⚠️ Type `/unpinall confirm` to unpin EVERYTHING everywhere."
		}
		return helperGlobalUnpin(bot)

	case "/setupDB":
		if !CheckAdminRightsSuper(userID) {
			LogInfo("⚠️ UNAUTHERIZED: DB Setup attempt by user %d", userID)
			return "⛔ Super Admin only."
		}
		if err := DBCreateTables(); err != nil {
			return fmt.Sprintf("❌ DB Setup Error: %v", err)
		}
		return "✅ Database tables created/verified successfully!"
	}

	return ""
}

// --- LOGIC HELPERS ---

func helperResolveTarget(update *tgbotapi.Update, args []string) (int64, string) {
	if update.Message.ReplyToMessage != nil {
		return update.Message.ReplyToMessage.From.ID, update.Message.ReplyToMessage.From.UserName
	}
	if len(args) > 1 {
		// Try to look up by name in DB
		if id := DBFindUserID(args[1]); id != 0 {
			return id, args[1]
		}
	}
	return update.Message.From.ID, update.Message.From.UserName
}

func helperBroadcastAndPin(bot *tgbotapi.BotAPI, text, pinner string) string {
	groups, err := DBGetKnownGroups()
	if err != nil {
		return "❌ DB Error getting groups."
	}

	// !! - GOROUTINE -!! //
	// Create channels to handel requests concurrently
	jobs := make(chan int64, len(groups))
	results := make(chan bool, len(groups))

	// Worker function - starts 3 workers !!! DO NOT EXCEED TELEGRAM RATE LIMITS (30/s) !!!
	for workerInstance := 1; workerInstance <= 3; workerInstance++ {
		go func(id int) {
			for chatID := range jobs {
				// Send
				msg, err := bot.Send(tgbotapi.NewMessage(chatID, text))
				if err == nil {
					// Pin
					bot.Request(tgbotapi.PinChatMessageConfig{ChatID: chatID, MessageID: msg.MessageID})
					// Save to DB
					DBPinMessage(chatID, msg.MessageID, pinner, text)
					results <- true
				} else {
					results <- false
				}
			}
		}(workerInstance)
	}

	// Add jobs
	for _, id := range groups {
		jobs <- id
	}
	close(jobs)

	// Collect results
	successCount := 0
	for i := 0; i < len(groups); i++ {
		if <-results {
			successCount++
		}
	}

	LogInfo("BROADCAST MESSAGE SENT:\nSent to %d/%d groups.", successCount, len(groups))

	return fmt.Sprintf("📢 Broadcast complete! Sent to %d/%d groups.", successCount, len(groups))
}

func helperGlobalUnpin(bot *tgbotapi.BotAPI) string {
	groups, err := DBGetKnownGroups()
	if err != nil {
		return "❌ DB Error getting groups during GLOBAL UNPIN operation."
	}
	// !! - GOROUTINE -!! //
	// Create channels to handel requests concurrently
	jobs := make(chan int64, len(groups))
	results := make(chan bool, len(groups))

	// Worker function - starts 3 workers !!! DO NOT EXCEED TELEGRAM RATE LIMITS (30/s) !!!
	for workerInstance := 1; workerInstance <= 3; workerInstance++ {
		go func(id int) {
			for chatID := range jobs {
				// API Unpin All
				_, err := bot.Request(tgbotapi.UnpinAllChatMessagesConfig{ChatID: chatID})

				// We clear the DB regardless of API success.
				// If the bot was kicked, we still want to remove the "ghost" pins from our DB.
				DBUnpinAllInChat(chatID)

				// Report success only if API call worked
				results <- (err == nil)
			}
		}(workerInstance)
	}

	// Add jobs
	for _, id := range groups {
		jobs <- id
	}
	close(jobs)

	// Collect results
	successCount := 0
	for i := 0; i < len(groups); i++ {
		if <-results {
			successCount++
		}
	}
	LogInfo("BROADCAST MESSAGE SENT:\n🌍 Global Unpin Complete.\n✅ Success: %d groups\n⚠️ Failed/Skipped: %d groups", successCount, len(groups)-successCount)

	return fmt.Sprintf("🌍 Global Unpin Complete.\n✅ Success: %d groups\n⚠️ Failed/Skipped: %d groups", successCount, len(groups)-successCount)
}
