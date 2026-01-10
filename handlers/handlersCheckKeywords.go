package handlers

import (
	"fmt"
	"regexp"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Shared Struct
type Keyword struct {
	ID         int            `json:"id"`
	Pattern    string         `json:"pattern"`
	IsNegative bool           `json:"is_negative"`
	Regex      *regexp.Regexp `json:"-"`
	AddedBy    string         `json:"added_by"`
}

// Memory Cache
var keywordCache []Keyword

// ReloadKeywords fetches from DB and compiles regex (Called on Startup & Add/Delete)
func ReloadKeywords() error {
	rawList, err := DBLoadKeywords()
	if err != nil {
		return err
	}

	var compiled []Keyword
	for _, k := range rawList {
		// Compile Regex
		re, err := regexp.Compile(k.Pattern)
		if err != nil {
			LogError("Skipping invalid regex pattern ID %d: %v", k.ID, err)
			continue
		}
		k.Regex = re
		compiled = append(compiled, k)
	}

	keywordCache = compiled
	LogInfo("🔄 Keywords Reloaded! Total Active: %d", len(keywordCache))
	return nil
}

// CheckMessageKeywords checks if a message matches any cached regex
func CheckMessageKeywords(update *tgbotapi.Update) string {
	if update.Message == nil {
		return ""
	}
	text := update.Message.Text

	// 1. Check for Keyword Match
	var matchedKeyword Keyword
	matched := false

	for _, kw := range keywordCache {
		if kw.Regex.MatchString(text) {
			matched = true
			matchedKeyword = kw
			break
		}
	}
	if !matched {
		return ""
	}

	// 2. Determine the Target User
	var targetID int64
	var targetName string

	// PRIORITY A: Is it a Reply?
	if update.Message.ReplyToMessage != nil {
		targetUser := update.Message.ReplyToMessage.From
		targetID = targetUser.ID
		targetName = targetUser.UserName
		if targetName == "" {
			targetName = targetUser.FirstName
		}

	} else {
		// PRIORITY B: Is there a @Mention?
		// Split text into words to find the first valid @username
		words := strings.Fields(text)
		for _, word := range words {
			if strings.HasPrefix(word, "@") {
				// Clean punctuation (e.g. "Thanks @User!" -> "@User")
				cleanWord := strings.TrimRight(word, ".,!?:;")

				// Look up this username in our Database
				foundID := DBFindUserID(cleanWord)
				if foundID != 0 {
					targetID = foundID
					targetName = cleanWord
					break
				}
			}
		}
	}

	// 3. If no target found, stop here
	if targetID == 0 {
		return ""
	}

	// 4. Anti-Farming: Prevent users from repping themselves
	if update.Message.From.ID == targetID {
		return fmt.Sprintf("⛔ @%s, you cannot increase your own reputation!", update.Message.From.UserName)
	}

	// 5. Apply Reputation Change
	var newRep int
	var err error
	var response string

	if matchedKeyword.IsNegative {
		newRep, err = DBDecreaseReputation(targetID, targetName)
		response = fmt.Sprintf("📉 %s -1 Rep (Trigger: '%s')\nTotal: %d", helperEnsureAtPrefix(targetName), matchedKeyword.Pattern, newRep)
	} else {
		newRep, err = DBAddReputation(targetID, targetName)
		response = fmt.Sprintf("🌟 %s +1 Rep!\nTotal: %d", helperEnsureAtPrefix(targetName), newRep)
	}

	if err != nil {
		LogError("DB Rep Update Failed: %v", err)
		return "⛔ Database Error."
	}
	return response
}

func formatName(user *tgbotapi.User) string {
	if user.UserName != "" {
		return "@" + user.UserName
	}
	return user.FirstName
}

func helperEnsureAtPrefix(name string) string {
	if strings.HasPrefix(name, "@") {
		return name
	}
	return "@" + name
}
