package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// PinEntry represents a tracked pinned message
type PinEntry struct {
	InternalID  int    `json:"internal_id"`
	ChatID      int64  `json:"chat_id"`
	MessageID   int    `json:"message_id"`
	PinnedBy    string `json:"pinned_by"`
	TextSnippet string `json:"text_snippet"`
}

var (
	pinMap      = make(map[int]PinEntry)
	knownGroups = make(map[int64]bool) // Tracks groups for unpinning
	pinFile     = "handlers\\maps\\mapsPinMessages.json"
	groupFile   = "handlers\\maps\\mapsPinGroups.json"
	nextPinID   = 1
	pinMutex    sync.Mutex
)

// -- PIN LOGIC -- //

// Pins the specific message in the chat and saves the entry to the DB
func PinMessage(bot *tgbotapi.BotAPI, chatID int64, messageID int, pinner string, text string) string {
	pinMutex.Lock()
	defer pinMutex.Unlock()

	// Send pin API request
	pinConfig := tgbotapi.PinChatMessageConfig{
		ChatID:    chatID,
		MessageID: messageID,
	}
	// Sends ERROR if the request to pin failed
	if _, err := bot.Request(pinConfig); err != nil {
		LogError("Failed to PIN message in Chat %d: %v", chatID, err)
		return fmt.Sprintf("⚠️ ERROR: Failed to pin message: %v", err)
	}

	// Save to DB if successful for tracking
	id := nextPinID
	nextPinID++

	// Creates a text snippet for ease of use
	snippet := text
	if len(snippet) > 50 {
		snippet = snippet[:47] + "..."
	}
	if snippet == "" {
		snippet = "[No Text/ Media]"
	}

	pinMap[id] = PinEntry{
		InternalID:  id,
		ChatID:      chatID,
		MessageID:   messageID,
		PinnedBy:    pinner,
		TextSnippet: snippet,
	}

	helperSavePins()
	return fmt.Sprintf("✅ Message Pinned and Saved to DB! (Pin ID: #%d)", id)
}

// Unpins a message by its Internal ID
func UnpinByID(bot *tgbotapi.BotAPI, id int) string {
	pinMutex.Lock()
	defer pinMutex.Unlock()

	entry, exists := pinMap[id]
	if !exists {
		return fmt.Sprintf("⚠️ ERROR: Pin ID #%d not found.", id)
	}

	// Send unpin API request
	unpinConfig := tgbotapi.UnpinChatMessageConfig{
		ChatID:    entry.ChatID,
		MessageID: entry.MessageID,
	}
	// Sends ERROR if the request to unpin failed
	if _, err := bot.Request(unpinConfig); err != nil {
		LogError("Failed to UNPIN message %d: %v", entry.MessageID, err)
		return fmt.Sprintf("⚠️ ERROR: Failed to unpin (maybe it was already deleted?): %v", err)
	}

	// Delete DB entry
	delete(pinMap, id)
	helperSavePins()

	return fmt.Sprintf("🗑️ Unpinned message #%d.", id)
}

// Unpins everything in the current group where the command was sent
func UnpinAllInChat(bot *tgbotapi.BotAPI, chatID int64) string {
	pinMutex.Lock()
	defer pinMutex.Unlock()

	// Send unpin all pins API request
	// NOTE: This unpins all pins - tracked and untracked
	// Sends ERROR if the request to unpin failed
	if _, err := bot.Request(tgbotapi.UnpinAllChatMessagesConfig{ChatID: chatID}); err != nil {
		return fmt.Sprintf("⚠️ ERROR:Could not unpin all messages: %v", err)
	}

	// Clear our DB for this specific ChatID
	// We have to loop because our map is Keyed by ID, not ChatID
	for id, entry := range pinMap {
		if entry.ChatID == chatID {
			delete(pinMap, id)
		}
	}
	helperSavePins()

	return "🗑️ All messages in this group have been unpinned in this group."
}

func UnpinAllGlobal(bot *tgbotapi.BotAPI) string {
	pinMutex.Lock()
	// Copy targets to avoid holding lock during network calls
	targets := make([]int64, 0, len(knownGroups))
	for chatID := range knownGroups {
		targets = append(targets, chatID)
	}
	pinMutex.Unlock()

	countSuccess := 0
	countFail := 0

	for _, chatID := range targets {
		// 4. Reuse existing logic for single chat
		// UnpinAllInChat handles its own locking and database persistence
		result := UnpinAllInChat(bot, chatID)

		// 5. Track success/failure based on the text response
		// (Your UnpinAllInChat returns "⚠️ ERROR..." on failure)
		if strings.Contains(result, "ERROR") {
			countFail++
		} else {
			countSuccess++
		}
	}

	return fmt.Sprintf("🌍 Global Unpin Complete.\n✅ Success: %d groups\n⚠️ Failed: %d groups", countSuccess, countFail)
}

// Sends a message to all known groups and pins it
func BroadcastAndPin(bot *tgbotapi.BotAPI, baseText string, pinner string) string {
	pinMutex.Lock()
	// We must be careful with locking. We need to iterate groups and modifying map.
	// To be safe, we'll copy the group list then process.
	targets := make([]int64, 0, len(knownGroups))
	for chatID := range knownGroups {
		targets = append(targets, chatID)
	}
	pinMutex.Unlock() // Unlock so we can do slow network calls

	count := 0

	for _, chatID := range targets {
		pinMutex.Lock()
		// Reserve an ID for this specific group's message
		currentID := nextPinID
		nextPinID++
		pinMutex.Unlock()

		// 1. Append ID to text
		finalText := fmt.Sprintf("%s\n\n📌 Pin ID: #%d", baseText, currentID)

		// 2. Send Message
		msg := tgbotapi.NewMessage(chatID, finalText)
		sentMsg, err := bot.Send(msg)

		if err == nil {
			// 3. Pin It
			pinConfig := tgbotapi.PinChatMessageConfig{ChatID: chatID, MessageID: sentMsg.MessageID}
			bot.Request(pinConfig)

			// 4. Save to DB (We manually save here to use the Pre-Reserved ID)
			pinMutex.Lock()

			// Create Snippet
			snippet := baseText
			if len(snippet) > 30 {
				snippet = snippet[:27] + "..."
			}

			pinMap[currentID] = PinEntry{
				InternalID:  currentID,
				ChatID:      chatID,
				MessageID:   sentMsg.MessageID,
				PinnedBy:    pinner,
				TextSnippet: snippet,
			}
			helperSavePins()
			pinMutex.Unlock()

			count++
		}
	}
	return fmt.Sprintf("📢 Broadcasted and pinned to %d groups.", count)
}

// -- HELPERS -- //

// -- Tracking -- //
// Adds a group to the known list for /pinall
func HelperRegisterGroup(chatID int64) {
	pinMutex.Lock()
	defer pinMutex.Unlock()

	if _, exists := knownGroups[chatID]; !exists {
		knownGroups[chatID] = true
		helperSaveGroups()
	}
}

func HelperListPins(chatID int64) string {
	pinMutex.Lock()
	defer pinMutex.Unlock()

	// Collect pins for this chat
	var list []PinEntry
	for _, pin := range pinMap {
		if pin.ChatID == chatID {
			list = append(list, pin)
		}
	}

	if len(list) == 0 {
		return "No active pins tracked in this chat."
	}

	// Sort by ID
	sort.Slice(list, func(i, j int) bool {
		return list[i].InternalID < list[j].InternalID
	})

	// Build String
	out := "📌 **Active Pins**:\n\n"
	for _, pin := range list {
		out += fmt.Sprintf("ID #%d: \"%s\" (by %s)\n\n", pin.InternalID, pin.TextSnippet, pin.PinnedBy)
	}
	return out
}

// -- Data Persistence -- //
func helperSavePins() {
	list := make([]PinEntry, 0, len(pinMap))
	for _, p := range pinMap {
		list = append(list, p)
	}
	data, _ := json.MarshalIndent(list, "", "  ")
	os.WriteFile(pinFile, data, 0664)
}

func helperSaveGroups() {
	// Convert map keys to slice
	list := make([]int64, 0, len(knownGroups))

	for id := range knownGroups {
		list = append(list, id)
	}

	data, _ := json.MarshalIndent(list, "", "  ")
	os.WriteFile(groupFile, data, 0664)
}

// LoadPinManager loads both Pins and Groups
func LoadPinManager() {
	// Load Pins
	if data, err := os.ReadFile(pinFile); err == nil {
		var list []PinEntry
		if json.Unmarshal(data, &list) == nil {
			maxID := 0
			for _, p := range list {
				pinMap[p.InternalID] = p
				if p.InternalID > maxID {
					maxID = p.InternalID
				}
			}
			nextPinID = maxID + 1
		}
	}

	// Load Groups
	if data, err := os.ReadFile(groupFile); err == nil {
		var list []int64
		if json.Unmarshal(data, &list) == nil {
			for _, id := range list {
				knownGroups[id] = true
			}
		}
	}
}
