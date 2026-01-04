package handlers

import (
	"os"
	"strconv"
	"strings"
)

// Checks if a user is in the authorised list of admins for admin-commands - checks Env.env for ID
func CheckAdminRights(userID int64) bool {
	// Gets ID string from Env.env
	adminEnv := os.Getenv("ADMIN_ID")

	if adminEnv == "" {
		LogError("WARNING: no ADMIN_IDs found in the environment variables doc - Admin level commands are inaccessible:")
		return false
	}

	// Split env strings by commas
	adminIDs := strings.Split(adminEnv, ",")

	// Loop through list of IDs to check if userID matches adminID
	for _, adminIdString := range adminIDs {
		// Strip whitespace
		adminIdString = strings.TrimSpace(adminIdString)

		// Convert env ID to int64 - (store in var, from base10, to int64)
		TrimmedAdminIdString, err := strconv.ParseInt(adminIdString, 10, 64)
		if err != nil {
			LogError("WARNING: Invalid Admin ID in Env.env: %s", adminIdString)
			continue
		}

		if TrimmedAdminIdString == userID {
			return true
		}
	}
	return false
}
