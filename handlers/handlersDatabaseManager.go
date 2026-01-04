package handlers

import (
	"database/sql"
	"os"

	_ "github.com/lib/pq" // Postgres driver
)

var DataBase *sql.DB

// --- INTIALISE DB --- //
func InitDB() {
	connStr := os.Getenv("DB_CONNECTION_STRING")

	if connStr == "" {
		// LOG AND CRASH - CRITICAL
		LogError("❌ FATAL: DB_CONNECTION_STRING is not set in Env.env")
		os.Exit(1)
	}

	var err error
	DataBase, err = sql.Open("postgres", connStr)

	if err != nil {
		// LOG AND CRASH - CRITICAL
		LogError("❌ FATAL: Could not open DB connection: %v", err)
		os.Exit(1)
	}

	// Verify connection
	if err = DataBase.Ping(); err != nil {
		LogError("❌ FATAL: Could not reach DB container. Is Podman running? Error: %v", err)
		os.Exit(1)
	}

	// Optimisation : connection pooling
	DataBase.SetMaxOpenConns(25)
	DataBase.SetMaxIdleConns(5)
	// Creates the tables if they don't exist yet
	createTables()

	LogInfo("✅ Database Connected & Tables Ready!")
}

func createTables() {
	queries := []string{
		// 1. Reputation Table
		`CREATE TABLE IF NOT EXISTS reputation (
			user_id BIGINT PRIMARY KEY,
			username TEXT NOT NULL,
			score INT DEFAULT 0
		);`,
		// 2. Keywords Table
		`CREATE TABLE IF NOT EXISTS keywords (
			id SERIAL PRIMARY KEY,
			pattern TEXT NOT NULL,
			is_negative BOOLEAN DEFAULT FALSE,
			added_by TEXT
		);`,
		// 3. Pins Table
		`CREATE TABLE IF NOT EXISTS pins (
			internal_id SERIAL PRIMARY KEY,
			chat_id BIGINT NOT NULL,
			message_id INT NOT NULL,
			pinned_by TEXT,
			text_snippet TEXT
		);`,
		// 4. Pin Groups Table (for /pinall)
		`CREATE TABLE IF NOT EXISTS pin_groups (
			chat_id BIGINT PRIMARY KEY
		);`,
	}

	for _, query := range queries {
		_, err := DataBase.Exec(query)
		if err != nil {
			LogError("❌ Database Table Creation Failed:\nQuery: %s\nError: %v", query, err)
		}
	}
}

func DBAddReputation(userID int64, username string) (int, error) {
	username = helperEnsureAtPrefix(username)

	// Upsert: Insert if new (Start at 1), Update if exists (Add 1)
	query := `
		INSERT INTO reputation (user_id, username, score)
		VALUES ($1, $2, 1)
		ON CONFLICT (user_id) 
		DO UPDATE SET score = reputation.score + 1;
	`

	// We pass only 2 arguments: userID ($1) and username ($2)
	// The '1' is hardcoded in the query above.
	_, err := DataBase.Exec(query, userID, username)

	if err != nil {
		return 0, err
	}
	return DBGetReputationSCore(userID)
}

func DBGetReputationSCore(userID int64) (int, error) {
	var score int

	err := DataBase.QueryRow("SELECT score FROM reputation Where user_id = $1", userID).Scan(&score)

	if err == sql.ErrNoRows {
		// If user doesnt exist yet, return 0 (not an error)
		return 0, nil
	}
	return score, err
}
