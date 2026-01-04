package handlers

import (
	"fmt"
	"regexp"

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

	// Iterate over CACHE, not DB (for speed)
	var matchedKeyword Keyword
	matched := false

	for _, kw := range keywordCache {
		if kw.Regex.MatchString(text) {
			matched = true
			matchedKeyword = kw
			break
		}
	}

	// Logic: If Matched AND is a Reply
	if matched && update.Message.ReplyToMessage != nil {
		fromUser := update.Message.From
		toUser := update.Message.ReplyToMessage.From

		if fromUser.ID == toUser.ID {
			return fmt.Sprintf("⛔ @%s, you cannot increase your own reputation!", fromUser.UserName)
		}

		var newRep int
		var err error
		var response string

		if matchedKeyword.IsNegative {
			newRep, err = DBDecreaseReputation(toUser.ID, toUser.UserName)
			response = fmt.Sprintf("📉 %s -1 Reputation! (Trigger: '%s')\nTotal Rep: %d", formatName(toUser), matchedKeyword.Pattern, newRep)
		} else {
			newRep, err = DBAddReputation(toUser.ID, toUser.UserName)
			response = fmt.Sprintf("🌟 %s +1 Reputation!\nTotal Rep: %d", formatName(toUser), newRep)
		}

		if err != nil {
			LogError("DB Rep Update Failed: %v", err)
			return "⛔ Database Error."
		}
		return response
	}
	return ""
}

func formatName(user *tgbotapi.User) string {
	if user.UserName != "" {
		return "@" + user.UserName
	}
	return user.FirstName
}
