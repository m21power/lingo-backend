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
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// helper struct for the query below
type candidate struct {
	ID       int64
	Username string
}
type trio struct {
	u1, u2, u3 candidate // u3.ID == 0 when not used
}

// GenerateDailyPairs groups every waiting user into pairs
// and runs once per day (e.g. scheduled at 06:00).
func GenerateDailyPairs(db *sql.DB) error {

	ctx := context.Background()
	opt := option.WithCredentialsFile("lingo-firestore.json")

	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return err
	}
	// 1. Fetch pairing candidates from Firebase
	users, err := fetchAllUsers(app)
	if err != nil {
		return err
	}
	if len(users) == 0 {
		log.Println("No users to pair today.")
		return nil
	}

	// 2️⃣ Shuffle for fairness
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(users), func(i, j int) { users[i], users[j] = users[j], users[i] })

	// 3️⃣ Build pair definitions in memory
	var groups []trio
	for i := 0; i < len(users); i += 2 {
		// If we only have one left, add him/her to previous pair
		if i+1 >= len(users) {
			if len(groups) == 0 { // impossible unless only 1 user total
				groups = append(groups, trio{u1: candidate{ID: users[i].UserID, Username: users[i].Username}})
			} else {
				groups[len(groups)-1].u3 = candidate{ID: users[i].UserID, Username: users[i].Username}
			}
			break
		}
		groups = append(groups, trio{
			u1: candidate{ID: users[i].UserID, Username: users[i].Username},
			u2: candidate{ID: users[i+1].UserID, Username: users[i+1].Username},
		})
	}

	// 4️⃣ Insert inside a single Tx
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // safe even on success

	insertPair := `
	INSERT INTO pairs
	    (user1id, user2id, user3id,
	     username1, username2, username3,
	     date, status)
	VALUES ($1,$2,$3,$4,$5,$6,CURRENT_DATE,'pending')
	RETURNING id`
	insertPart := `
	INSERT INTO pair_participation
	    (pair_id, userid)
	VALUES ($1,$2)`

	for _, g := range groups {
		sortTrio(&g) // Sort user IDs ascending before insert

		var pairID int64
		if err := tx.QueryRowContext(ctx, insertPair,
			g.u1.ID, g.u2.ID, g.u3.ID,
			g.u1.Username, g.u2.Username, g.u3.Username,
		).Scan(&pairID); err != nil {
			return fmt.Errorf("insert pair: %w", err)
		}

		// insert user1 + user2
		if _, err := tx.ExecContext(ctx, insertPart, pairID, g.u1.ID); err != nil {
			return err
		}
		if g.u2.ID != 0 {
			if _, err := tx.ExecContext(ctx, insertPart, pairID, g.u2.ID); err != nil {
				return err
			}
		}
		if g.u3.ID != 0 { // special‑group member
			if _, err := tx.ExecContext(ctx, insertPart, pairID, g.u3.ID); err != nil {
				return err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	log.Printf("✅ Generated %d pair(s) for %s\n", len(groups), time.Now().Format("2006‑01‑02"))
	return nil
}

// helper to sort trio by ID ascending
func sortTrio(t *trio) {
	type user struct {
		ID       int64
		Username string
	}

	users := []user{
		{t.u1.ID, t.u1.Username},
		{t.u2.ID, t.u2.Username},
	}

	// Only add u3 if present (ID != 0)
	if t.u3.ID != 0 {
		users = append(users, user{t.u3.ID, t.u3.Username})
	}

	// Sort by ID ascending
	sort.Slice(users, func(i, j int) bool {
		return users[i].ID < users[j].ID
	})

	// Reassign sorted values back to trio
	t.u1.ID, t.u1.Username = users[0].ID, users[0].Username
	t.u2.ID, t.u2.Username = users[1].ID, users[1].Username
	if len(users) == 3 {
		t.u3.ID, t.u3.Username = users[2].ID, users[2].Username
	} else {
		t.u3.ID, t.u3.Username = 0, ""
	}
}

type FirebaseUser struct {
	UserID   int64  `firestore:"userId"`
	Username string `firestore:"username"`
}

func fetchAllUsers(app *firebase.App) ([]FirebaseUser, error) {
	ctx := context.Background()
	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	iter := client.Collection("users").Documents(ctx)

	var users []FirebaseUser
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var user FirebaseUser
		if err := doc.DataTo(&user); err != nil {
			continue // skip invalid entries
		}
		users = append(users, user)
	}

	return users, nil
}
