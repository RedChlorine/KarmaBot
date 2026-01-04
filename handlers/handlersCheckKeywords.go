package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Keyword struct {
	ID         int            `json:"id"`
	Pattern    string         `json:"pattern"`
	IsNegative bool           `json:"is_negative"`
	Regex      *regexp.Regexp `json:"-"`
	AddedBy    string         `json:"added_by"`
}

var (
	keywords     = make(map[int]Keyword)
	nextKeyWord  = 1
	keywordsFile = "handlers/maps/mapsKeywords.json"
)

// CheckMessageKeywords checks if a message contains a keyword and handles Reputation replies
// Called from main.go to process all messages
// SUB-CRITICAL FUNC
func CheckMessageKeywords(update *tgbotapi.Update) string {
	if update.Message == nil {
		return ""
	}

	text := update.Message.Text

	// Check if the message matches any of the defined keywords
	var matchedKeyword Keyword // Stores the keyword for processing
	matched := false
	for _, keyword := range keywords {
		if keyword.Regex.MatchString(text) {
			matched = true
			matchedKeyword = keyword
			break
		}
	}

	// IF matched == true && is a reply
	if matched && update.Message.ReplyToMessage != nil {
		fromUser := update.Message.From
		toUser := update.Message.ReplyToMessage.From

		// GUARD: Prevents user from altering their own rep
		if fromUser.ID == toUser.ID {
			return fmt.Sprintf("⛔ @%s, you cannot increase your own reputation!", fromUser.UserName)
		}

		var newReputation int
		var responseMessage string
		var err error

		// IF NEG keyword => DECREMENT
		if matchedKeyword.IsNegative {
			newReputation, err = DecreaseReputation(toUser.ID, toUser.UserName)
			if err != nil {
				LogError("DB Save Failed (Decrease Rep) for %s: %v", toUser.UserName, err)
				return "⛔ERROR: Could not update reputation Databse: Inform an admin.⛔"
			}

			// Write response message
			responseMessage = fmt.Sprintf("📉 %s -1 Reputation! (Trigger: '%s')\nTotal Rep: %d", formatName(toUser), matchedKeyword.Pattern, newReputation)

		} else {
			// IF POS keyword => INCREMENT (DEFAULT ACTION)
			newReputation, err = AddReputation(toUser.ID, toUser.UserName)
			if err != nil {
				LogError("DB Save Failed (Increase Rep) for %s: %v", toUser.UserName, err)
				return "⛔ERROR: Could not update reputation Databse: Inform an admin.⛔"
			}

			responseMessage = fmt.Sprintf("🌟 %s +1 Reputation!\nTotal Rep: %d", formatName(toUser), newReputation)
		}
		return responseMessage
	}
	return ""
}

/*
* /addkeyword command
* DEFAULT => POSITIVE KW
 */
func AddKeyword(pattern string, update *tgbotapi.Update) string {
	return addKeywordInternal(pattern, false, update)
}

/*
* /addnegkeyword command
* DEFAULT => POSITIVE KW
 */
func AddNegativeKeyword(pattern string, update *tgbotapi.Update) string {
	return addKeywordInternal(pattern, true, update)
}

// Internal helper to avoid duplicate code
func addKeywordInternal(pattern string, isNegative bool, update *tgbotapi.Update) string {
	pattern = strings.TrimSpace(pattern)

	if pattern == "" {
		return "Error: Pattern cannot be empty."
	}

	// Check for dupliacte keywords
	for _, kw := range keywords {
		if kw.Pattern == pattern {
			return fmt.Sprintf("ERROR: Keyword already exists as #%d", kw.ID)
		}
	}

	// Compile regex if regex string was passed
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Sprintf("ERROR: Invalid regex: %v", err)
	}

	username := update.Message.From.UserName
	if username == "" {
		username = fmt.Sprintf("%d", update.Message.From.ID)
	}

	id := nextKeyWord
	nextKeyWord++

	// Create struct for keyword
	keywords[id] = Keyword{
		ID:         id,
		Pattern:    pattern,
		IsNegative: isNegative,
		Regex:      regex,
		AddedBy:    username,
	}

	if err := saveKeywordsToFile(); err != nil {
		return fmt.Sprintf("ERROR: Failed to save keyword to disk: %v", err)
	}

	typeLabel := "POSITIVE"
	if isNegative {
		typeLabel = "NEGATIVE"
	}

	return fmt.Sprintf("✅ Added %s keyword #%d: '%s'", typeLabel, id, pattern)

}

/*
* Called from handlersCheckCommands - deletes a keyword by ID
* /deletekeyword
 */
func DeleteKeyword(idString string) string {
	id, err := strconv.Atoi(idString)
	if err != nil {
		return "Error: Keyword ID must be a number."
	}

	if _, ok := keywords[id]; !ok {
		return fmt.Sprintf("Error: No keyword with the ID %d.", id)
	}

	delete(keywords, id)

	if err := saveKeywordsToFile(); err != nil {
		return fmt.Sprintf("Deleted keyword #%d but failed to save to disk: %v", id, err)
	}

	return fmt.Sprintf("Deleted keyword #%d.", id)
}

/*
* Called from handlersCheckCommands - lists all the keywords in memory
* /listkeywords
 */
func ListKeywords() string {
	if len(keywords) == 0 {
		return "⛔Error: No keywords have been defined⛔"
	}

	// Convert map to slice for sorting
	var list []Keyword

	for _, kw := range keywords {
		list = append(list, kw)
	}

	// Sort the slice: ID Descending (Highest to Lowest)
	sort.Slice(list, func(i, j int) bool {
		return list[i].ID > list[j].ID
	})

	// Build the output string
	var keywordList strings.Builder
	keywordList.WriteString("📜 **Current Keywords** (Newest First):\n")

	// Output keyword list
	for _, kw := range list {
		// Add a visual indicator for negative words
		tag := "➕"
		if kw.IsNegative {
			tag = "⛔ [NEG]"
		}

		fmt.Fprintf(&keywordList, "#%d %s: \"%s\" (by @%s)\n", kw.ID, tag, kw.Pattern, kw.AddedBy)
	}
	return keywordList.String()
}

// Saves keywords to handlers/maps/mapsKeywords.json for persistance
func saveKeywordsToFile() error {
	// Flatten the map to a slice for stable JSON
	list := make([]Keyword, 0, len(keywords))

	for _, kw := range keywords {
		list = append(list, kw)
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(keywordsFile, data, 0664)
}

// Loads the keywords from handlers/maps/mapsKeywords.json and reconstructs the list of allowed keywords
func LoadKeywordFromFile() error {
	data, err := os.ReadFile(keywordsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no file yet
		}
		return err
	}

	var list []Keyword
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}

	keywords = make(map[int]Keyword)
	maxID := 0

	for _, kw := range list {
		re, err := regexp.Compile(kw.Pattern)
		if err != nil {
			//skip bad patterns
			continue
		}
		kw.Regex = re
		keywords[kw.ID] = kw
		if kw.ID > maxID {
			maxID = kw.ID
		}
	}
	nextKeyWord = maxID + 1
	return nil
}

// Helper to format name for response
func formatName(user *tgbotapi.User) string {
	if user.UserName != "" {
		return "@" + user.UserName
	}
	return user.FirstName
}
