package handlers

import (
	"fmt"

	_ "github.com/lib/pq" // Postgres driver
)

func DBCreateTables() error {
	// --- CHECK IF DB IS EMPTY --- //
	// - Check if Reputation table exists
	var exists bool
	checkQuery := "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'reputation');"
	err := DB.QueryRow(checkQuery).Scan(&exists)

	if err != nil {
		LogError("❌ ERROR: Failed to check if reputation table exists %v", err) // Log Channel
		return fmt.Errorf("❌ ERROR: Failed to check if reputation table exists %v", err)
	}

	// !!! -CRITICAL TO PREVENT DATA LOSS- !!!
	// If the tables exist - return early
	if exists {
		err := fmt.Errorf("Database tables already exist | Table creation stopped...")
		LogError("⚠️ **WARNING:\nDATABASE DETECTED** ⚠️\n\n%s \n\n(If this is a migration, verify the DB_CONNECTION_STRING!", err)
		return err
	}

	// Create Tables
	queries := []string{
		`CREATE TABLE IF NOT EXISTS reputation (
			user_id BIGINT PRIMARY KEY,
			username TEXT NOT NULL,
			score INT DEFAULT 0
		);`,
		`CREATE TABLE IF NOT EXISTS keywords (
			id SERIAL PRIMARY KEY,
			pattern TEXT NOT NULL,
			is_negative BOOLEAN DEFAULT FALSE,
			added_by TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS pins (
			internal_id SERIAL PRIMARY KEY,
			chat_id BIGINT NOT NULL,
			message_id INT NOT NULL,
			pinned_by TEXT,
			text_snippet TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS pin_groups (
			chat_id BIGINT PRIMARY KEY
		);`,
		`CREATE TABLE IF NOT EXISTS bot_admins(
			user_id BIGINT PRIMARY KEY,
			username TEXT, 
			role TEXT NOT NULL DEFAULT 'admin' -- 'admin' or 'superadmin'
		);`,
	}

	for _, query := range queries {
		_, err := DB.Exec(query)
		if err != nil {
			LogError("❌ERROR: Database Table Creation Failed:\nQuery: %s\nError: %v", query, err)
			return err
		}
	}
	return nil
}
