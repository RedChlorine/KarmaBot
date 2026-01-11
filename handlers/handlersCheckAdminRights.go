package handlers

import (
	"log"
	"os"
	"strconv"
)

// Checks if a user is in the authorised list of admins for super-commands - checks DB for ID
func CheckAdminRightsSuper(userID int64) bool {
	// --- PRIORITY CHECK: DB --- //
	var role string
	err := DB.QueryRow("SELECT role FROM bot_admins WHERE user_id = $1", userID).Scan(&role)

	if err != nil {
		log.Printf("❌ DB ERROR: Could not parse superadmin rights for %d from the DB - FALLBACK to Config.env: %v", userID, err)
		LogError("⚠️⚠️⚠️ FALLBACK TRIGGERED ⚠️⚠️⚠️: Could not parse superadmin rights for %d from the DB - FALLBACK to Config.env", userID)

		// --- FALLBACK CHECK: Config File (prevent lockout) --- //
		headAdminIDStr := os.Getenv("HEAD_ADMIN_ID")
		if headAdminIDStr != "" {

			headAdminID, err := strconv.ParseInt(headAdminIDStr, 10, 64)
			if err != nil {
				log.Panicf("[!!PANIC!!]\nHEAD_ADMIN_ID could not be determined! - check config.env for empty strings! - LOCKOUT RISK!")
				return false
			}
			if userID == headAdminID {
				// Optional: Auto-fix the DB if the config admin is missing
				go func() {
					username := os.Getenv("HEAD_ADMIN_USERNAME")
					DBAddAdmin(headAdminID, username, "superadmin")
				}()
				return role == "superadmin" // Authorized via Config File
			}
		}
	}
	return role == "superadmin"
}

// Checks if a user is in the authorised list of admins for admin-commands - checks DB for ID
func CheckAdminRights(userID int64) bool {
	var role string
	err := DB.QueryRow("SELECT role FROM bot_admins WHERE user_id = $1", userID).Scan(&role)
	if err != nil {
		LogError("❌ DB ERROR: Could not parse admin rights for %d from the DB: %v", userID, err)
		return false
	}
	// Superadmins are also admins
	return role == "admin" || role == "superadmin"
}
