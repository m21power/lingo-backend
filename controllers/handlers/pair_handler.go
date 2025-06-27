package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"lingo-backend/usecase"
	util "lingo-backend/utils"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	firebase "firebase.google.com/go/v4"

	"github.com/gorilla/mux"
	"google.golang.org/api/option"
)

type PairHandler struct {
	usecase usecase.PairUsecase
}

func NewPairHandler(usecase usecase.PairUsecase) *PairHandler {
	return &PairHandler{
		usecase: usecase,
	}
}

func (p *PairHandler) GetDailyPairs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["userId"]
	userId, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		http.Error(w, "Invalid userId query parameter", http.StatusBadRequest)
		return
	}
	pair, err := p.usecase.GetDailyPairs(userId)
	if err != nil {
		http.Error(w, "Error fetching daily pairs", http.StatusInternalServerError)
		return
	}
	if pair.ID == 0 {
		http.Error(w, "No pairs found for today", http.StatusNotFound)
		return
	}

	var yes []User

	// Case: 3-member group (special group)
	if pair.User3ID != 0 {
		if pair.User1Participating && pair.User2Participating && pair.User3Participating {
			yes = append(yes,
				User{pair.User1ID, pair.Username1},
				User{pair.User2ID, pair.Username2},
				User{pair.User3ID, pair.Username3},
			)
		}
	} else {
		// Case: 2-member pair
		if pair.User1Participating && pair.User2Participating {
			yes = append(yes,
				User{pair.User1ID, pair.Username1},
				User{pair.User2ID, pair.Username2},
			)
		}
	}

	ctx := context.Background()
	config := &firebase.Config{
		DatabaseURL: "https://lingo-19e2a-default-rtdb.firebaseio.com/",
	}
	app, err := firebase.NewApp(ctx, config, option.WithCredentialsFile("lingo-firestore.json"))
	if err != nil {
		log.Println("Firebase init error:", err)
	} else {
		if len(yes) >= 2 {
			isSpecial := len(yes) == 3
			err := saveChatroomToRealtimeDB(app, yes, isSpecial)
			if err != nil {
				log.Println("Failed to save chatroom:", err)
			}
		}

	}

	util.WriteJSON(w, http.StatusOK, pair)
}

func (p *PairHandler) UpdatePairParticipation(w http.ResponseWriter, r *http.Request) {
	type Payload struct {
		PairID        string `json:"pairId"`
		UserID        int64  `json:"userId"`
		Participating bool   `json:"participating"`
	}
	var payload Payload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		util.WriteError(w, err, http.StatusBadRequest)
		return
	}
	err = p.usecase.UpdatePairParticipation(payload.PairID, payload.UserID, payload.Participating)
	if err != nil {
		util.WriteError(w, err, http.StatusBadRequest)
		return
	}
	util.WriteJSON(w, http.StatusOK, "Updated Successfully!")
}

type User struct {
	ID       int64
	Username string
}

// Save chatroom to Firebase Realtime DB
func saveChatroomToRealtimeDB(app *firebase.App, yes []User, isSpecialGroup bool) error {
	ctx := context.Background()

	sort.Slice(yes, func(i, j int) bool {
		return yes[i].ID < yes[j].ID
	})

	ids := make([]string, len(yes))
	usernames := make([]string, len(yes))
	seen := make(map[string]bool)

	for i, u := range yes {
		ids[i] = fmt.Sprint(u.ID)
		usernames[i] = u.Username
		seen[fmt.Sprint(u.ID)] = false
	}

	roomID := strings.Join(ids, "_")

	message := fmt.Sprintf("You have been paired with @%s today!", usernames[1])
	if isSpecialGroup && len(usernames) == 3 {
		message = fmt.Sprintf("You are in a special group with @%s and @%s today!", usernames[1], usernames[2])
	}

	chatroom := map[string]interface{}{
		"chatid":               roomID,
		"participantUsernames": usernames,
		"participantIds":       ids,

		"message":          message,
		"senderid":         "admin",
		"sendername":       "Admin Bot",
		"senderprofilepic": "https://example.com/bot.png",
		"createdat":        time.Now().Format(time.RFC3339),
		"seen":             seen,
		"isspecialgroup":   isSpecialGroup,
	}

	client, err := app.Database(ctx)
	if err != nil {
		return err
	}

	ref := client.NewRef("chats/" + roomID)
	if err := ref.Set(ctx, chatroom); err != nil {
		return fmt.Errorf("failed to save chatroom: %w", err)
	}

	log.Println("Chatroom saved to Realtime DB:", roomID)
	return nil
}
