package handlers

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
)

// Struct representing UserReputation for a specific user
type UserReputation struct {
	UserID   int64  `json:"user_id"`
	UserName string `json:"username"`
	UserRep  int    `json:"user_reputation"`
}

var (
	reputationMap  = make(map[int64]*UserReputation)
	reputationFile = "handlers/maps/mapsReputation.json"
	repMutex       sync.Mutex
)

// AddReputation increments a user's rep and saves to disk
func AddReputation(userID int64, username string) (int, error) {
	repMutex.Lock()
	defer repMutex.Unlock()

	// If user exists, increment score - prevents added rep to deleted accounts
	if userReputation, ok := reputationMap[userID]; ok {
		userReputation.UserRep++

		// Update username in case they change it
		if username != "" {
			userReputation.UserName = username
		}
	} else {
		// Create new user entry to file
		reputationMap[userID] = &UserReputation{
			UserID:   userID,
			UserName: username,
			UserRep:  1,
		}
	}
	return reputationMap[userID].UserRep, saveReputationToFile()
}

// DecreaseReputation removes 1 rep (Command: /decrep)
func DecreaseReputation(userID int64, username string) (int, error) {
	repMutex.Lock()
	defer repMutex.Unlock()

	if userRep, ok := reputationMap[userID]; ok {
		userRep.UserRep--
		if username != "" {
			userRep.UserName = username
		}
	} else {
		// If user doesn't exist, start them at -1
		reputationMap[userID] = &UserReputation{
			UserID:   userID,
			UserName: username,
			UserRep:  -1,
		}
	}
	return reputationMap[userID].UserRep, saveReputationToFile()
}

// SetReputation forces a specific score (Command: /setrep)
func SetReputation(userID int64, username string, val int) (int, error) {
	repMutex.Lock()
	defer repMutex.Unlock()

	if userRep, ok := reputationMap[userID]; ok {
		userRep.UserRep = val
		if username != "" {
			userRep.UserName = username
		}
	} else {
		reputationMap[userID] = &UserReputation{
			UserID:   userID,
			UserName: username,
			UserRep:  val,
		}
	}
	return val, saveReputationToFile()
}

// GetReputation returns the score and name of a user. Returns -1 if not found.
func GetReputation(userID int64) (int, string) {
	repMutex.Lock()
	defer repMutex.Unlock()

	if user, ok := reputationMap[userID]; ok {
		return user.UserRep, user.UserName
	}

	return 0, "ERROR: Reputation of User not found"
}

// saveReputationToFile saves the map to JSON
func saveReputationToFile() error {
	list := make([]UserReputation, 0, len(reputationMap))
	for _, rep := range reputationMap {
		list = append(list, *rep)
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(reputationFile, data, 0664)
}

// LoadReputationFromFile loads data into memory on startup
func LoadReputationFromFile() error {
	data, err := os.ReadFile(reputationFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No file yet, that's fine
		}
		return err
	}

	var list []UserReputation
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}

	repMutex.Lock()
	defer repMutex.Unlock()
	reputationMap = make(map[int64]*UserReputation)
	for i := range list {
		reputationMap[list[i].UserID] = &list[i]
	}
	return nil
}

/*********************HELPERS**************************/

func HelperFindUserID(username string) int64 {
	repMutex.Lock()
	defer repMutex.Unlock()

	cleanName := strings.TrimPrefix(username, "@")
	for _, user := range reputationMap {
		if strings.EqualFold(user.UserName, cleanName) {
			return user.UserID
		}
	}
	return 0
}
