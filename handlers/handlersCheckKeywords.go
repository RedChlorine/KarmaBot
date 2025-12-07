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
	keywordsFile = "handlers\\maps\\mapsKeywords.json"
)

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

// Called from main loop for normal messages
func CheckMessageKeywords(update *tgbotapi.Update) string {
	if update.Message == nil {
		return ""
	}
	text := update.Message.Text
	for _, kw := range keywords {
		if kw.Regex.MatchString(text) {
			return "Great"
		}
	}
	return ""
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
