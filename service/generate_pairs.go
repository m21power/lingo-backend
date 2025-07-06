package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"time"

	firebase "firebase.google.com/go/v4"
	db "firebase.google.com/go/v4/db"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type candidate struct {
	ID         int64
	Username   string
	ProfileURL string
}
type trio struct {
	u1, u2, u3 *candidate // u3 can be nil
}

func GenerateDailyPairs(db *sql.DB, rtdbClient *db.Client) (string, error) {
	ctx := context.Background()
	opt := option.WithCredentialsFile("lingo-firestore.json")
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return "", err
	}

	users, err := fetchAllUsers(app)
	if err != nil || len(users) == 0 {
		log.Println("🚫 No users to pair today.")
		return "", nil
	}

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(users), func(i, j int) { users[i], users[j] = users[j], users[i] })

	var groups []trio
	for i := 0; i < len(users); i += 2 {
		if i+1 >= len(users) {
			// Last odd user → make trio with previous pair
			if len(groups) > 0 {
				groups[len(groups)-1].u3 = &users[i]
			} else {
				// Only one user total
				groups = append(groups, trio{u1: &users[i]})
			}
			break
		}
		groups = append(groups, trio{
			u1: &users[i],
			u2: &users[i+1],
		})
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	for _, g := range groups {
		// Sort by ID for uniqueness
		cleanAndSort(&g)

		pairID := fmt.Sprintf("%d_%d", g.u1.ID, g.u2.ID)
		if g.u3 != nil {
			pairID += fmt.Sprintf("_%d", g.u3.ID)
		}

		_, err := tx.ExecContext(ctx, `
			INSERT INTO pairs (id, user1id, user2id, user3id,
				username1, username2, username3, date, status)
			VALUES ($1,$2,$3,$4,$5,$6,$7,CURRENT_DATE,'pending')`,
			pairID, g.u1.ID, g.u2.ID, idOrZero(g.u3),
			g.u1.Username, g.u2.Username, usernameOrEmpty(g.u3))
		if err != nil {
			return "", fmt.Errorf("insert pair failed: %w", err)
		}

		// Participation
		for _, u := range []*candidate{g.u1, g.u2, g.u3} {
			if u != nil {
				if _, err := tx.ExecContext(ctx,
					`INSERT INTO pair_participation (pair_id, userid) VALUES ($1,$2)`, pairID, u.ID); err != nil {
					return "", fmt.Errorf("insert participation: %w", err)
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}
	log.Printf("✅ %d group(s) created for %s\n", len(groups), time.Now().Format("2006-01-02"))

	for _, g := range groups {
		pairID := fmt.Sprintf("%d_%d", g.u1.ID, g.u2.ID)
		if g.u3 != nil {
			pairID += fmt.Sprintf("_%d", g.u3.ID)
		}
		if err := PushPairToRealtimeDB(ctx, rtdbClient, pairID, g); err != nil {
			return "", fmt.Errorf("failed to push pair %s to Realtime DB: %w", pairID, err)
		}
		log.Printf("✅ Pair %s pushed to Realtime DB\n", pairID)
	}
	return fmt.Sprintf("✅ %d group(s) created for %s", len(groups), time.Now().Format("2006-01-02")), nil
}

// Sorts trio by ID
func cleanAndSort(t *trio) {
	var all []*candidate
	all = append(all, t.u1)
	if t.u2 != nil {
		all = append(all, t.u2)
	}
	if t.u3 != nil {
		all = append(all, t.u3)
	}

	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })

	t.u1, t.u2, t.u3 = all[0], nil, nil
	if len(all) > 1 {
		t.u2 = all[1]
	}
	if len(all) > 2 {
		t.u3 = all[2]
	}
}

func idOrZero(c *candidate) int64 {
	if c == nil {
		return 0
	}
	return c.ID
}
func usernameOrEmpty(c *candidate) string {
	if c == nil {
		return ""
	}
	return c.Username
}

type FirebaseUser struct {
	UserID     int64  `firestore:"userId"`
	Username   string `firestore:"username"`
	ProfileURL string `firestore:"profileUrl"`
}

func fetchAllUsers(app *firebase.App) ([]candidate, error) {
	ctx := context.Background()
	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	iter := client.Collection("users").Documents(ctx)

	var users []candidate
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var u FirebaseUser
		if err := doc.DataTo(&u); err != nil {
			continue
		}
		users = append(users, candidate{
			ID: u.UserID, Username: u.Username, ProfileURL: u.ProfileURL,
		})
	}
	return users, nil
}

func PushPairToRealtimeDB(ctx context.Context, client *db.Client, pairID string, g trio) error {
	// Collect participants
	participants := []*candidate{g.u1, g.u2}
	if g.u3 != nil {
		participants = append(participants, g.u3)
	}

	// Prepare participant data
	var usernames []string
	var ids []int64
	var images []string
	unreadCounts := map[string]int{}
	for _, u := range participants {
		usernames = append(usernames, u.Username)
		ids = append(ids, u.ID)
		images = append(images, u.ProfileURL)
		unreadCounts[fmt.Sprintf("%d", u.ID)] = 1
	}

	// 1️⃣ Create chat metadata
	chatData := map[string]interface{}{
		"name":                 "Special Group",
		"isGroup":              len(participants) == 3,
		"participantUsernames": usernames,
		"participantIds":       ids,
		"participantImages":    images,
		"lastMessage":          "You've been paired for today's conversation!",
		"lastMessageTime":      time.Now().UnixMilli(),
		"seenBy":               []string{},
		"unreadCounts":         unreadCounts,
		"generatedAt":          time.Now().UnixMilli(),
	}

	chatRef := client.NewRef("chats/" + pairID)
	if err := chatRef.Set(ctx, chatData); err != nil {
		return fmt.Errorf("failed to write chat: %w", err)
	}

	// 2️⃣ Add system message
	message := map[string]interface{}{
		"text":            "You've been paired for today's conversation!",
		"senderName":      "admin",
		"senderId":        1,
		"createdAt":       time.Now().UnixMilli(),
		"seenBy":          []string{},
		"isSystemMessage": true,
		"isParticipating": []string{},
	}

	msgRef, err := client.NewRef("messages/"+pairID).Push(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create message reference: %w", err)
	}
	if err := msgRef.Set(ctx, message); err != nil {
		return fmt.Errorf("failed to push system message: %w", err)
	}

	// 3️⃣ Link chat to each user's chat list
	for _, u := range participants {
		userChatPath := fmt.Sprintf("userChats/%d/%s", u.ID, pairID)
		if err := client.NewRef(userChatPath).Set(ctx, true); err != nil {
			return fmt.Errorf("failed to link chat to user %d: %w", u.ID, err)
		}
	}

	return nil
}
