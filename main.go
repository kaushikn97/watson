package main

import (
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/exp/slices"
)

var AllCharacters = []string{"plum", "mustard", "orchid", "scarlett", "green", "peacock"}
var AllRooms = []string{"kitchen", "dining", "hall", "conservatory", "study", "billiards", "library", "lounge"}
var AllWeapons = []string{"rope", "pipe", "revolver", "dagger", "candlestick", "wrench"}

type suggestion struct {
	Character string `json:"character"`
	Room      string `json:"room"`
	Weapon    string `json:"weapon"`
}

type logEntry struct {
	Suggester    string     `json:"suggester"`
	Suggestion   suggestion `json:"suggestion"`
	Responder    string     `json:"responder"`
	RevealedCard string     `json:"revealedCard"`
	Id           string     `json:"id"`
}

func ValidateLogEntry(logEntry *logEntry, session *session) error {
	// Implement
	return nil
}

func addLogEntry(logEntry *logEntry, session *session) error {

	session.Log = append(session.Log, *logEntry)
	fmt.Println(session.Log)
	if logEntry.RevealedCard != "" {
		updateSoTOnCardReveal(logEntry.RevealedCard, logEntry.Responder, session)
	} else {
		RunAllLogs(session)
	}
	return nil
}

func RunAllLogs(Session *session) {
	updated := false
	for _, logEntry := range Session.Log {
		updated = updated || updateSoTOnLogEntry(&logEntry, Session)
	}
	if updated {
		RunAllLogs(Session)
	}
}

type sourceOfTruth struct {
	CharacterMap map[string]map[string]int `json:"characterMap"`
	RoomMap      map[string]map[string]int `json:"roomMap"`
	WeaponMap    map[string]map[string]int `json:"weaponMap"`
}

func updateSoTOnLogEntry(logEntry *logEntry, Session *session) bool {
	// what if there is no response?
	suggesterIdx := slices.IndexFunc(Session.Players, func(playerName string) bool { return playerName == logEntry.Suggester })
	responderIdx := slices.IndexFunc(Session.Players, func(playerName string) bool { return playerName == logEntry.Responder })
	playerIdx := suggesterIdx
	initPlayerIdx := playerIdx
	playerName := Session.Players[playerIdx]
	updated := false
	if responderIdx != -1 {
		for playerIdx != responderIdx {
			if Session.SoT.CharacterMap[playerName][logEntry.Suggestion.Character] == 0 {
				Session.SoT.CharacterMap[playerName][logEntry.Suggestion.Character] = -1
				updated = true
			}
			if Session.SoT.RoomMap[playerName][logEntry.Suggestion.Room] == 0 {
				Session.SoT.RoomMap[playerName][logEntry.Suggestion.Room] = -1
				updated = true
			}
			if Session.SoT.WeaponMap[playerName][logEntry.Suggestion.Weapon] == 0 {
				Session.SoT.WeaponMap[playerName][logEntry.Suggestion.Weapon] = -1
				updated = true
			}
			playerIdx = (playerIdx + 1) % len(Session.Players)
			playerName = Session.Players[playerIdx]
		}
	}

	if responderIdx == -1 {
		first := true
		for playerIdx != initPlayerIdx || first {
			if Session.SoT.CharacterMap[playerName][logEntry.Suggestion.Character] == 0 {
				Session.SoT.CharacterMap[playerName][logEntry.Suggestion.Character] = -1
				updated = true
			}
			if Session.SoT.RoomMap[playerName][logEntry.Suggestion.Room] == 0 {
				Session.SoT.RoomMap[playerName][logEntry.Suggestion.Room] = -1
				updated = true
			}
			if Session.SoT.WeaponMap[playerName][logEntry.Suggestion.Weapon] == 0 {
				Session.SoT.WeaponMap[playerName][logEntry.Suggestion.Weapon] = -1
				updated = true
			}
			playerIdx = (playerIdx + 1) % len(Session.Players)
			playerName = Session.Players[playerIdx]
			first = false
		}
	}

	if responderIdx != -1 {
		if Session.SoT.CharacterMap[playerName][logEntry.Suggestion.Character] < 0 && Session.SoT.RoomMap[playerName][logEntry.Suggestion.Room] < 0 {
			if Session.SoT.WeaponMap[playerName][logEntry.Suggestion.Weapon] == 0 {
				Session.SoT.WeaponMap[playerName][logEntry.Suggestion.Weapon] = 1
				updated = true
			}
			for _, player := range Session.Players {
				if player != playerName {
					if Session.SoT.WeaponMap[player][logEntry.Suggestion.Weapon] == 0 {
						Session.SoT.WeaponMap[player][logEntry.Suggestion.Weapon] = -1
						updated = true
					}
				}
			}
		} else if Session.SoT.WeaponMap[playerName][logEntry.Suggestion.Weapon] < 0 && Session.SoT.RoomMap[playerName][logEntry.Suggestion.Room] < 0 {
			if Session.SoT.CharacterMap[playerName][logEntry.Suggestion.Character] == 0 {
				Session.SoT.CharacterMap[playerName][logEntry.Suggestion.Character] = 1
				updated = true
			}
			for _, player := range Session.Players {
				if player != playerName {
					if Session.SoT.RoomMap[player][logEntry.Suggestion.Character] == 0 {
						Session.SoT.CharacterMap[player][logEntry.Suggestion.Character] = -1
						updated = true
					}
				}
			}
		} else if Session.SoT.CharacterMap[playerName][logEntry.Suggestion.Character] < 0 && Session.SoT.WeaponMap[playerName][logEntry.Suggestion.Weapon] < 0 {
			if Session.SoT.RoomMap[playerName][logEntry.Suggestion.Room] == 0 {
				Session.SoT.RoomMap[playerName][logEntry.Suggestion.Room] = 1
				updated = true
			}
			for _, player := range Session.Players {
				if player != playerName {
					if Session.SoT.RoomMap[player][logEntry.Suggestion.Room] == 0 {
						Session.SoT.RoomMap[player][logEntry.Suggestion.Room] = -1
						updated = true
					}
				}
			}
		}
	}
	for _, character := range AllCharacters {
		found := false
		for _, player := range Session.Players {
			if Session.SoT.CharacterMap[player][character] == 1 {
				found = true
				break
			}
		}
		if found {
			for _, player := range Session.Players {
				if Session.SoT.CharacterMap[player][character] == 0 {
					Session.SoT.CharacterMap[player][character] = -1
					updated = true
				}
			}
		}
	}
	for _, room := range AllRooms {
		found := false
		for _, player := range Session.Players {
			if Session.SoT.RoomMap[player][room] == 1 {
				found = true
				break
			}
		}
		if found {
			for _, player := range Session.Players {
				if Session.SoT.RoomMap[player][room] == 0 {
					Session.SoT.RoomMap[player][room] = -1
					updated = true
				}
			}
		}
	}
	for _, weapon := range AllWeapons {
		found := false
		for _, player := range Session.Players {
			if Session.SoT.WeaponMap[player][weapon] == 1 {
				found = true
				break
			}
		}
		if found {
			for _, player := range Session.Players {
				if Session.SoT.WeaponMap[player][weapon] == 0 {
					Session.SoT.WeaponMap[player][weapon] = -1
					updated = true
				}
			}
		}
	}
	return updated
}

func updateSoTOnCardReveal(revealedCard string, responder string, Session *session) {
	updated := false
	if slices.Contains(AllCharacters, revealedCard) {
		if Session.SoT.CharacterMap[responder][revealedCard] != 1 {
			updated = true
		}
		Session.SoT.CharacterMap[responder][revealedCard] = 1
	} else if slices.Contains(AllRooms, revealedCard) {
		if Session.SoT.RoomMap[responder][revealedCard] != 1 {
			updated = true
		}
		Session.SoT.RoomMap[responder][revealedCard] = 1
	} else if slices.Contains(AllWeapons, revealedCard) {
		if Session.SoT.WeaponMap[responder][revealedCard] != 1 {
			updated = true
		}
		Session.SoT.WeaponMap[responder][revealedCard] = 1
	}

	if updated {
		RunAllLogs(Session)
	}

}

func ValidateRevealedCard(revealedCard string, session *session) error {
	// Implement
	return nil
}

func ValidateRevealedCharacterCard(revealedCard string, session *session) error {
	// Implement
	return nil
}

func ValidateRevealedRoomCard(revealedCard string, session *session) error {
	// Implement
	return nil
}

func ValidateRevealedWeaponCard(revealedCard string, session *session) error {
	// Implement
	return nil
}

type session struct {
	Log        []logEntry
	Id         string
	Players    []string
	SoT        sourceOfTruth
	MainPlayer string
}

var lock = &sync.Mutex{}

type sessionManager struct {
	Sessions map[string]*session
}

func getSession(sessionId string) (*session, error) {
	session, exists := GetSessionManager().Sessions[sessionId]
	if exists {
		return session, nil
	} else {
		return nil, errors.New("session id not found")
	}
}

var SessionManager *sessionManager

func GetSessionManager() *sessionManager {
	if SessionManager != nil {
		fmt.Println("Session manager already created.")
	} else {
		lock.Lock()
		defer lock.Unlock()
		if SessionManager != nil {
			fmt.Println("Session manager already created.")
		} else {
			fmt.Println("Creating session manager instance now.")
			SessionManager = &sessionManager{}
			SessionManager.Sessions = make(map[string]*session)
		}
	}
	return SessionManager
}

func NewSession(players []string, mainPlayer string) *session {

	session := session{}
	session.SoT.CharacterMap = make(map[string]map[string]int)
	session.SoT.RoomMap = make(map[string]map[string]int)
	session.SoT.WeaponMap = make(map[string]map[string]int)
	session.MainPlayer = mainPlayer

	for _, value := range players {
		session.Players = append(session.Players, value)
		inputValue := 0
		if value == session.MainPlayer {
			inputValue = -1
		}

		session.SoT.CharacterMap[value] = make(map[string]int)
		for _, character := range AllCharacters {
			session.SoT.CharacterMap[value][character] = inputValue
		}

		session.SoT.RoomMap[value] = make(map[string]int)
		for _, room := range AllRooms {
			session.SoT.RoomMap[value][room] = inputValue
		}

		session.SoT.WeaponMap[value] = make(map[string]int)
		for _, weapon := range AllWeapons {
			session.SoT.WeaponMap[value][weapon] = inputValue
		}

	}
	return &session
}

type createSessionRequest struct {
	Players    []string `json:"players"`
	MainPlayer string   `json:"mainPlayer"`
}

type createSessionResponse struct {
	SessionId string `json:"sessionId"`
}

func createSessionHandler(c *gin.Context) {
	var createSessionRequest createSessionRequest
	fmt.Println(c.Request.Body)
	if err := c.BindJSON(&createSessionRequest); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}
	sessionId := uuid.New().String()
	GetSessionManager().Sessions[sessionId] = NewSession(createSessionRequest.Players, createSessionRequest.MainPlayer)
	c.IndentedJSON(http.StatusCreated, createSessionResponse{SessionId: sessionId})
}

type addLogEntryRequest struct {
	LogEntry logEntry `json:"logEntry"`
}

func addLogEntryHandler(c *gin.Context) {
	var addLogEntryRequest addLogEntryRequest

	if err := c.BindJSON(&addLogEntryRequest); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}

	sessionId := c.Param("sessionId")

	session, err := getSession(sessionId)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid session id"})
		return
	}

	if err := ValidateLogEntry(&(addLogEntryRequest.LogEntry), session); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid log entry"})
		return
	}

	if err := ValidateRevealedCard(addLogEntryRequest.LogEntry.RevealedCard, session); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid revealed card"})
		return
	}
	addLogEntryRequest.LogEntry.Id = uuid.New().String()
	addLogEntry(&(addLogEntryRequest.LogEntry), session)

	c.IndentedJSON(http.StatusCreated, session.Log)
}

type addCardsRequest struct {
	RevealedCharacters []string `json:"revealedCharacters"`
	RevealedRooms      []string `json:"revealedRooms"`
	RevealedWeapons    []string `json:"revealedWeapons"`
}

type addCardsResponse struct {
	SessionId string `json:"sessionId"`
}

func addCardsHandler(c *gin.Context) {
	var addCardsRequest addCardsRequest

	if err := c.BindJSON(&addCardsRequest); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}

	sessionId := c.Param("sessionId")

	session, err := getSession(sessionId)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid session id"})
		return
	}

	for _, character := range addCardsRequest.RevealedCharacters {
		if err := ValidateRevealedCharacterCard(character, session); err != nil {
			c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid revealed character card"})
			return
		}
		updateSoTOnCardReveal(character, session.MainPlayer, session)
	}
	for _, room := range addCardsRequest.RevealedRooms {
		if err := ValidateRevealedRoomCard(room, session); err != nil {
			c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid revealed room card"})
			return
		}
		updateSoTOnCardReveal(room, session.MainPlayer, session)
	}
	for _, weapon := range addCardsRequest.RevealedWeapons {
		if err := ValidateRevealedWeaponCard(weapon, session); err != nil {
			c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid revealed weapon card"})
			return
		}
		updateSoTOnCardReveal(weapon, session.MainPlayer, session)
	}
	c.IndentedJSON(http.StatusCreated, addCardsResponse{SessionId: sessionId})
}

func sourceOfTruthHandler(c *gin.Context) {
	sessionId := c.Param("sessionId")

	session, err := getSession(sessionId)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid session id"})
		return
	}
	c.IndentedJSON(http.StatusOK, session.SoT)

}

func playerListHandler(c *gin.Context) {
	sessionId := c.Param("sessionId")

	session, err := getSession(sessionId)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid session id"})
		return
	}
	c.IndentedJSON(http.StatusOK, session.Players)

}

func logEntries(c *gin.Context) {
	sessionId := c.Param("sessionId")
	session, err := getSession(sessionId)

	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message": "Invalid session id"})
		return
	}
	c.IndentedJSON(http.StatusOK, session.Log)
}

func main() {
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"POST", "GET"},
	}))

	router.POST("/createSession", createSessionHandler)

	router.POST("/:sessionId/addLogEntry", addLogEntryHandler)

	router.POST("/:sessionId/addCards", addCardsHandler)

	router.GET("/:sessionId/sourceOfTruth", sourceOfTruthHandler)

	router.GET("/:sessionId/playerList", playerListHandler)

	router.GET("/:sessionId/logs", logEntries)

	router.Run("localhost:8080")
}
