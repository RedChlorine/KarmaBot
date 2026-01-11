package handlers

// Checks if a user is in the authorised list of admins for super-commands - checks DB for ID
func CheckAdminRightsSuper(userID int64) bool {
	var role string
	err := DB.QueryRow("SELECT role FROM bot_admins WHERE user_id = $1", userID).Scan(&role)
	if err != nil {
		LogError("❌ DB ERROR: Could not parse superadmin rights for %d from the DB: %v", userID, err)
		return false
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
