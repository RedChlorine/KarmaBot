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

	//DEBUG
	//log.Printf("\nADDREP USERNAME: %s", username)

	// If user exists, increment score - prevents added rep to deleted accounts
	if userReputation, ok := reputationMap[userID]; ok {
		userReputation.UserRep++

		// -- DEPRICATED --
		// Update username in case they change it
		/*if username != "" {
			userReputation.UserName = username
		}*/

	} else {
		// Create new user entry to file
		reputationMap[userID] = &UserReputation{
			UserID:   userID,
			UserName: ensureAtPrefix(username),
			UserRep:  1,
		}
	}
	return reputationMap[userID].UserRep, saveReputationToFile()
}

// DecreaseReputation removes 1 rep (Command: /decrement)
func DecreaseReputation(userID int64, username string) (int, error) {
	repMutex.Lock()
	defer repMutex.Unlock()
	//log.Printf("\nDECREP USERNAME: %s", username)

	if userRep, ok := reputationMap[userID]; ok {
		userRep.UserRep--

		// -- DEPRICATED --
		/*if username != "" {
			userRep.UserName = username
		}*/

	} else {
		// If user doesn't exist, start them at -1
		reputationMap[userID] = &UserReputation{
			UserID:   userID,
			UserName: ensureAtPrefix(username),
			UserRep:  -1,
		}
	}
	return reputationMap[userID].UserRep, saveReputationToFile()
}

// SetReputation forces a specific score (Command: /setrep)
func SetReputation(userID int64, username string, val int) (int, error) {
	repMutex.Lock()
	defer repMutex.Unlock()
	//log.Printf("\nSETREP USERNAME: %s", username)

	if userRep, ok := reputationMap[userID]; ok {
		userRep.UserRep = val

		// -- DEPRICATED --
		/*if username != "" {
			userRep.UserName = username
		}*/

	} else {
		reputationMap[userID] = &UserReputation{
			UserID:   userID,
			UserName: ensureAtPrefix(username),
			UserRep:  val,
		}
	}
	return reputationMap[userID].UserRep, saveReputationToFile()
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
	//log.Printf("\nENTERED HelperFindUserID\n\nHELPER - FIND_USER_ID\nUSERNAME PASSED: %s", username)
	repMutex.Lock()
	defer repMutex.Unlock()

	target := username
	//log.Printf("\nHELPER - FIND_USER_ID\nTARGET: %s", target)
	//IF TARGET DOESNT HAVE AN @, THEN FORCE ADD AN @
	if !strings.HasPrefix(target, "@") {
		target = "@" + target
		//log.Printf("\n\nHELPER - FIND_USER_ID\nTARGET DID NOT HAVE AN @ SUFFIX - ADDED @: %s", target)
	}

	for _, user := range reputationMap {

		//log.Printf("\n\nHELPER - FIND_USER_ID\nLOGGING RANGE user VS target\nUSER:%s\nTARGET:%s\n", user.UserName, target)

		if strings.EqualFold(user.UserName, target) {
			//log.Printf("\n\nHELPER - FIND_USER_ID\nUSER ID FROM SAN TARGET: %d", user.UserID)
			return user.UserID
		}
	}

	return 0
}

// Helper: Ensure a string starts with exactly one @
// Used when CREATING new users to prevent "@@Username" or "Username" (no @)
func ensureAtPrefix(name string) string {
	if strings.HasPrefix(name, "@") {
		return name
	}
	return "@" + name
}
