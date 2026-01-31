package handlers

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"

	_ "github.com/lib/pq" // Postgres driver
)

var (
	DB         *sql.DB
	userCache  = make(map[int64]string)
	cacheMutex sync.RWMutex
)

// --- INTIALISE DB --- //
func InitDB() {
	connStr := os.Getenv("DB_CONNECTION_STRING")

	if connStr == "" {
		// LOG AND CRASH - CRITICAL
		LogError("❌ FATAL: DB_CONNECTION_STRING is not set in Config.env")
		os.Exit(1)
	}

	var err error
	DB, err = sql.Open("postgres", connStr)

	if err != nil {
		// LOG AND CRASH - CRITICAL
		LogError("❌ FATAL: Could not open DB connection:\n%v", err)
		os.Exit(1)
	}

	// Verify connection
	if err = DB.Ping(); err != nil {
		LogError("❌ FATAL: Could not reach DB container. Is the Database running? \nError: %v", err)
		os.Exit(1)
	}

	// Optimisation : connection pooling
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)

	// --- CHECK ONLY & DONT CREATE TABLES --- //
	var exists bool
	checkQuery := "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'reputation');"
	err = DB.QueryRow(checkQuery).Scan(&exists)
	if err != nil {
		LogError("❌ ERROR: Failed to check if reputation table exists \n%v", err) // Log Channel
		os.Exit(1)
	}
	if !exists {
		LogError("⚠️ **DATABASE CONNECTED BUT EMPTY** ⚠️\n\n❌ FATAL: Tables not found. The bot won't start, database commands will fail.\n\n👉 **Run /setupdb to initialize the schema.**\n(If this is a migration, CHECK YOUR CONNECTION STRING!)")
	} else {
		LogInfo("✅ Database Connected & Tables Found.")
	}
}

// CloseDB closes the database connection safely.
func CloseDB() {
	if DB != nil {
		err := DB.Close()
		if err != nil {
			LogError("❌ Error closing database: \n%v", err)
		} else {
			LogInfo("🗄️ Database connection closed successfully.")
		}
	}
}

// --- OPTIMIZED USER CACHE --- //
func DBInitUserCache() {
	rows, err := DB.Query("SELECT user_id, username FROM reputation")
	if err != nil {
		LogError("⚠️ WARNING: Failed to load user cache: %v\n", err)
		return
	}
	defer rows.Close()

	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	count := 0
	for rows.Next() {
		var id int64
		var name string
		rows.Scan(&id, &name)
		userCache[id] = name
		count++
	}
	LogInfo("🧠 User Cache Loaded: %d users in memory.", count)
}

func DBEnsureUserExists(userID int64, username string) {
	username = helperEnsureAtPrefix(username)

	cacheMutex.RLock()
	cachedName, exists := userCache[userID]
	cacheMutex.RUnlock()

	if exists && cachedName == username {
		return
	}

	cacheMutex.Lock()
	userCache[userID] = username
	cacheMutex.Unlock()

	go func() {
		query := `
			INSERT INTO reputation (user_id, username, score)
			VALUES ($1, $2, 0)
			ON CONFLICT (user_id) 
			DO UPDATE SET username = $2;
		`
		_, err := DB.Exec(query, userID, username)
		if err != nil {
			LogError("Failed to register user %s: \n%v", username, err)
		}
	}()
}

// --- SQL | REPUTATION HANDLERS --- //
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
	_, err := DB.Exec(query, userID, username)

	if err != nil {
		return 0, err
	}
	return DBGetReputationScore(userID)
}

func DBDecreaseReputation(userID int64, username string) (int, error) {
	username = helperEnsureAtPrefix(username)
	query := `
	INSERT INTO reputation(user_id, username, score)
	VALUES($1, $2, -1)
	ON CONFLICT (user_id)
	DO UPDATE SET score = reputation.score -1;
	`
	_, err := DB.Exec(query, userID, username)
	if err != nil {
		return 0, err
	}
	return DBGetReputationScore(userID)
}

func DBSetReputation(userID int64, username string, val int) (int, error) {
	username = helperEnsureAtPrefix(username)
	query := `
	INSERT INTO reputation(user_id, username, score)
	VALUES ($1, $2, $3)
	ON CONFLICT (user_id)
	DO UPDATE SET score = $3;
	`
	_, err := DB.Exec(query, userID, username, val)
	return val, err
}

func DBGetReputationScore(userID int64) (int, error) {
	var score int

	err := DB.QueryRow("SELECT score FROM reputation Where user_id = $1", userID).Scan(&score)

	if err == sql.ErrNoRows {
		// If user doesnt exist yet, return 0 (not an error)
		return 0, nil
	}
	return score, err
}

func DBGetTop10() string {
	rows, err := DB.Query("SELECT username, score FROM reputation ORDER BY score DESC LIMIT 10")
	if err != nil {
		LogError("❌ ERROR - Failed to fetch Top 10:%v", err)
		return "Error fetching leaderboard."
	}
	defer rows.Close()

	var sb strings.Builder
	sb.WriteString("🏆 **Reputation Leaderboard** 🏆\n\n")

	i := 1
	for rows.Next() {
		var name string
		var score int
		rows.Scan(&name, &score)
		fmt.Fprintf(&sb, "%d. %s | %d\n", i, name, score)
		i++
	}
	return sb.String()
}

func DBFindUserID(username string) int64 {
	target := username
	if !strings.HasPrefix(target, "@") {
		target = "@" + target
	}
	var id int64
	err := DB.QueryRow("SELECT user_id FROM reputation WHERE lower(username) = lower($1)", target).Scan(&id)
	if err != nil {
		return 0
	}
	return id
}

// --- SQL | KEYWORD HANDLERS --- //
func DBAddKeyword(pattern string, isNegative bool, addedBy string) (int, error) {
	// --- Check Duplicates --- //
	var existingID int
	checkQuery := "SELECT id FROM keywords WHERE pattern = $1 LIMIT 1"
	err := DB.QueryRow(checkQuery, pattern).Scan(&existingID)

	// If no error occurs, it means a row was found (the pattern exists)
	if err == nil {
		LogError("A user attempted to insert a duplicate pattern into the DB: '%s'\nExisting ID:# %d\n\n...Update to DB rejected.", pattern, existingID)

		return 0, fmt.Errorf("\nThe keyword: '%s',already exists (ID #%d)", pattern, existingID)
	}

	// If it doesn't exist (sql.ErrNoRows), proceed with the insertion
	var id int
	insertQuery := `
		INSERT INTO keywords (pattern, is_negative, added_by)
		VALUES ($1, $2, $3) RETURNING id`

	err = DB.QueryRow(insertQuery, pattern, isNegative, addedBy).Scan(&id)
	return id, err
}

func DBDeleteKeyword(id int) error {
	result, err := DB.Exec("DELETE FROM keywords WHERE id = $1", id)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		return fmt.Errorf("No keyword found with ID %d", id)
	}
	LogInfo("Keyword with ID %d deleted from database.", id)
	return nil
}

func DBListKeywords() string {
	rows, err := DB.Query("SELECT id, pattern, is_negative, added_by FROM keywords ORDER BY id DESC")
	if err != nil {
		return "Error fetching keywords."
	}
	defer rows.Close()

	var sb strings.Builder
	sb.WriteString("📜 **Current Keywords** (Newest First):\n\n")

	for rows.Next() {
		var id int
		var pattern, addedBy string
		var isNeg bool
		rows.Scan(&id, &pattern, &isNeg, &addedBy)

		tag := "✅ [POS]"
		if isNeg {
			tag = "⛔ [NEG]"
		}
		fmt.Fprintf(&sb, "#%d  %s : \"%s\" (by @%s)\n", id, tag, pattern, addedBy)
	}
	return sb.String()
}

func DBLoadKeywords() ([]Keyword, error) {
	rows, err := DB.Query("SELECT id, pattern, is_negative, added_by FROM keywords")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Keyword
	for rows.Next() {
		var keyword Keyword
		rows.Scan(&keyword.ID, &keyword.Pattern, &keyword.IsNegative, &keyword.AddedBy)
		list = append(list, keyword)
	}
	return list, nil
}

// --- SQL | PIN HANDLERS --- //
func DBPinMessage(chatID int64, messageID int, pinner, text string) (int, error) {
	var id int
	if len(text) > 50 {
		text = text[:47] + "..."
	}

	err := DB.QueryRow(`
		INSERT INTO pins (chat_id, message_id, pinned_by, text_snippet)
		VALUES ($1, $2, $3, $4) RETURNING internal_id`,
		chatID, messageID, pinner, text).Scan(&id)
	return id, err
}

func DBUnpinByID(internalID int) (int64, int, error) {
	var chatID int64
	var msgID int
	err := DB.QueryRow("DELETE FROM pins WHERE internal_id = $1 RETURNING chat_id, message_id", internalID).Scan(&chatID, &msgID)
	return chatID, msgID, err
}

func DBUnpinAllInChat(chatID int64) error {
	_, err := DB.Exec("DELETE FROM pins WHERE chat_id = $1", chatID)
	return err
}

func DBRegisterGroup(chatID int64) {
	DB.Exec("INSERT INTO pin_groups (chat_id) VALUES ($1) ON CONFLICT DO NOTHING", chatID)
}

func DBGetKnownGroups() ([]int64, error) {
	rows, err := DB.Query("SELECT chat_id FROM pin_groups")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []int64
	for rows.Next() {
		var id int64
		rows.Scan(&id)
		groups = append(groups, id)
	}
	return groups, nil
}

func DBListPins(chatID int64) string {
	rows, err := DB.Query("SELECT internal_id, text_snippet, pinned_by FROM pins WHERE chat_id = $1 ORDER BY internal_id ASC", chatID)
	if err != nil {
		return "Error loading pins."
	}
	defer rows.Close()

	var sb strings.Builder
	sb.WriteString("📌 **Active Pins**:\n\n")
	hasRows := false
	for rows.Next() {
		hasRows = true
		var id int
		var text, pinner string
		rows.Scan(&id, &text, &pinner)
		fmt.Fprintf(&sb, "ID #%d: \"%s\" (by %s)\n\n", id, text, pinner)
	}
	if !hasRows {
		return "No active pins tracked in this chat."
	}
	return sb.String()
}

// --- SQL | ADMIN HANDLERS --- //
func DBAddAdmin(userID int64, username, role string) error {
	_, err := DB.Exec(`
		INSERT INTO bot_admins (user_id, username, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id)
		DO UPDATE SET role = $3;`,
		userID, helperEnsureAtPrefix(username), role)
	return err
}

func DBRemoveAdmin(userID int64) error {
	_, err := DB.Exec("DELETE FROM bot_admins WHERE user_id = $1", userID)
	return err
}

func DBListAdmins() string {
	rows, err := DB.Query("SELECT user_id, username, role FROM bot_admins ORDER BY role DESC")
	if err != nil {
		return "Error fetching admin list."
	}
	defer rows.Close()

	var sb strings.Builder
	sb.WriteString("🛡️ **Bot Admins** 🛡️\n\n")
	for rows.Next() {
		var username, role string
		var userID int64
		rows.Scan(&userID, &username, &role)
		fmt.Fprintf(&sb, "%s | %s | ID: %v\n\n", username, role, userID)
	}
	return sb.String()
}
