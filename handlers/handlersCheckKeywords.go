package handlers

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Keyword struct {
	ID      int            `json:"id"`
	Pattern string         `json:"pattern"`
	Regex   *regexp.Regexp `json:"-"`
	AddedBy string         `json:"added_by"`
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
	matched := false
	for _, keyword := range keywords {
		if keyword.Regex.MatchString(text) {
			matched = true
			break
		}
	}

	// IF matched == true && is a reply
	if matched && update.Message.ReplyToMessage != nil {
		fromUser := update.Message.From
		toUser := update.Message.ReplyToMessage.From

		// Validation : Mitigate users incrementing their own rep or bot rep
		if toUser.IsBot {
			return ""
		}

		if fromUser.ID == toUser.ID {
			return fmt.Sprintf("⛔ @%s, you cannot increase your own reputation!", fromUser.UserName)
		}

		// UPDATE REPUTATION
		newReputation, err := AddReputation(toUser.ID, toUser.UserName)
		if err != nil {
			return "ERROR: Could not update reputation database - please infrom an admin."
		}

		// Create a response string - "@user your rep +1, current rep[someRepInt]"
		userLabel := toUser.FirstName
		if toUser.UserName != "" {
			userLabel = "@" + toUser.UserName
		}
		return fmt.Sprintf("🌟 %s +1 Reputation!\nTotal Rep: %d", userLabel, newReputation)
	}
	return ""
}

/*
* Called from handlersCheckCommands - Adds a keyword and assigns the added and an ID
* /addkeyword command
 */
func AddKeyword(pattern string, update *tgbotapi.Update) string {
	pattern = strings.TrimSpace(pattern)

	// Check for empty pattern - mitigate empty keywords being added
	if pattern == "" {
		return "Error: Keyword pattern cannot be empty."
	}

	// Check if pattern already exists (exact same regex string)
	for _, kw := range keywords {
		if kw.Pattern == pattern {
			return fmt.Sprintf("Error: Keyword already exists as #%d: %s (by @%s)", kw.ID, kw.Pattern, kw.AddedBy)
		}
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Sprintf("Error: Invalid regex: %v", err)
	}

	username := update.Message.From.UserName
	if username == "" {
		username = fmt.Sprintf("%d", update.Message.From.ID)
	}

	id := nextKeyWord
	nextKeyWord++
	// Create Keyword struct
	keywords[id] = Keyword{ID: id, Pattern: pattern, Regex: regex, AddedBy: username}

	if err := saveKeywordsToFile(); err != nil {
		return fmt.Sprintf("Added keyword #%d but failed to save to disk: %v", id, err)
	}

	return fmt.Sprintf("Added keyword #%d: %s by @%s", id, pattern, username)
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
		return "Error: No keywords have been defined."
	}

	var keywordList strings.Builder

	keywordList.WriteString("Current keywords:\n")

	//generates list - #id of keyword, pattern of keyword, the @ of the adder
	for id, kw := range keywords {
		fmt.Fprintf(&keywordList, "#%d %s (by @%s)\n", id, kw.Pattern, kw.AddedBy)
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
