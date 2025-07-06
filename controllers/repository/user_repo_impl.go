package repository

import (
	"context"
	"database/sql"
	"fmt"
	"lingo-backend/domain"
	services "lingo-backend/service"
	util "lingo-backend/utils"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	"firebase.google.com/go/v4/db"
	"github.com/lib/pq"
)

type UserRepoImpl struct {
	db         *sql.DB
	firestore  *firestore.Client
	rtdbClient *db.Client
}

func NewUserRepo(db *sql.DB, firestore *firestore.Client, rtdbClient *db.Client) *UserRepoImpl {
	return &UserRepoImpl{
		db:         db,
		firestore:  firestore,
		rtdbClient: rtdbClient,
	}
}

func (r *UserRepoImpl) FillAttendance(userIds []int64) error {
	ctx := context.Background()

	// Update status in your SQL DB
	query := "UPDATE pairs SET status = 'completed' WHERE user1id = ANY($1) OR user2id = ANY($1) OR user3id = ANY($1)"
	_, err := r.db.Exec(query, pq.Int64Array(userIds))
	if err != nil {
		return err
	}

	// Get yesterday's date string in YYYY-MM-DD format
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	for _, userId := range userIds {
		docID := strconv.FormatInt(userId, 10)

		_, err := r.firestore.Collection("users").Doc(docID).Update(ctx, []firestore.Update{
			{Path: "attendance", Value: firestore.Increment(1)},
		})
		if err != nil {
			return err
		}

		consistencyDoc := r.firestore.Collection("consistency").Doc(docID).Collection("dates").Doc(yesterday)
		_, err = consistencyDoc.Set(ctx, map[string]interface{}{
			"score": 1,
		}, firestore.MergeAll)
	}

	return nil
}

func (r *UserRepoImpl) MissAttendance(userId int64) error {
	ctx := context.Background()
	docID := strconv.FormatInt(userId, 10)
	_, err := r.firestore.Collection("users").Doc(docID).Update(ctx, []firestore.Update{
		{Path: "missCount", Value: firestore.Increment(1)},
	})

	if err != nil {
		return err
	}

	// Get yesterday's date string in YYYY-MM-DD format
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	consistencyDoc := r.firestore.Collection("consistency").Doc(docID).Collection("dates").Doc(yesterday)
	_, err = consistencyDoc.Set(ctx, map[string]interface{}{
		"score": 0,
	}, firestore.MergeAll)

	return nil
}

func (r *UserRepoImpl) PairUser(userId int64, username, profileUrl string) (util.PairResponse, error) {
	ctx := context.Background()

	// Check if someone is already waiting
	var waitlistId, otherUserId int64
	var otherUsername, otherProfileUrl string

	row := r.db.QueryRow("SELECT id, userid, username, profileurl FROM waitlist ORDER BY createdAt LIMIT 1")
	err := row.Scan(&waitlistId, &otherUserId, &otherUsername, &otherProfileUrl)
	if err == sql.ErrNoRows {
		// No one waiting, insert this user into waitlist
		_, err := r.db.Exec("INSERT INTO waitlist (userid, username, profileurl) VALUES ($1, $2, $3)", userId, username, profileUrl)
		return util.PairResponse{Wait: true}, err
	} else if err != nil {
		return util.PairResponse{}, fmt.Errorf("failed to query waitlist: %w", err)
	}
	if otherUserId == userId {
		return util.PairResponse{Wait: true}, nil
	}
	//  Found someone! Remove them from waitlist
	_, err = r.db.Exec("DELETE FROM waitlist WHERE id = $1", waitlistId)
	if err != nil {
		return util.PairResponse{}, err
	}

	//  Create chatId based on sorted user IDs
	var chatId string
	if userId < otherUserId {
		chatId = fmt.Sprintf("%d_%d", userId, otherUserId)
	} else {
		chatId = fmt.Sprintf("%d_%d", otherUserId, userId)
	}

	//  Prepare chat data
	participants := []struct {
		ID         int64
		Username   string
		ProfileURL string
	}{
		{ID: userId, Username: username, ProfileURL: profileUrl},
		{ID: otherUserId, Username: otherUsername, ProfileURL: otherProfileUrl},
	}

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

	chatData := map[string]interface{}{
		"name":                 "Daily Match",
		"isGroup":              false,
		"participantUsernames": usernames,
		"participantIds":       ids,
		"participantImages":    images,
		"lastMessage":          "As per your request, you have been paired. Please check your messages.",
		"lastMessageTime":      time.Now().UnixMilli(),
		"seenBy":               []string{},
		"unreadCounts":         unreadCounts,
	}

	// Write to Firebase Realtime DB
	chatRef := r.rtdbClient.NewRef("chats/" + chatId)
	if err := chatRef.Set(ctx, chatData); err != nil {
		return util.PairResponse{}, fmt.Errorf("failed to write chat: %w", err)
	}

	// Add system message to messages
	message := map[string]interface{}{
		"text":            "You've been paired for today's conversation!",
		"senderName":      "admin",
		"senderId":        1,
		"createdAt":       time.Now().UnixMilli(),
		"seenBy":          []string{},
		"isSystemMessage": true,
		"isParticipating": []string{},
	}

	msgRef, err := r.rtdbClient.NewRef("messages/"+chatId).Push(ctx, nil)
	if err != nil {
		return util.PairResponse{}, fmt.Errorf("failed to create message reference: %w", err)
	}
	if err := msgRef.Set(ctx, message); err != nil {
		return util.PairResponse{}, fmt.Errorf("failed to push system message: %w", err)
	}

	//  Link chat to each user's chat list
	for _, u := range participants {
		userChatPath := fmt.Sprintf("userChats/%d/%s", u.ID, chatId)
		if err := r.rtdbClient.NewRef(userChatPath).Set(ctx, true); err != nil {
			return util.PairResponse{}, fmt.Errorf("failed to link chat to user %d: %w", u.ID, err)
		}
	}
	// we will be emitting through websocket here and also save the notification in notifications table
	que := "INSERT INTO notifications (id, user1id, user2id, message, createdat) VALUES ($1, $2, $3, $4, $5)"
	_, err = r.db.Exec(que, chatId, userId, otherUserId, "You've been paired for today's conversation!", time.Now())
	if err != nil {
		return util.PairResponse{}, fmt.Errorf("failed to insert notification: %w", err)
	}
	return util.PairResponse{Wait: false}, nil
}

func (r *UserRepoImpl) GetNotifications(userId int64) (domain.NotificationResponse, error) {
	query := `
SELECT 
    n.id,
    n.user1id,
    n.user2id,
    n.message,
    n.createdat,
    CASE 
        WHEN ns.userId IS NULL THEN false
        ELSE true
    END AS seen
FROM notifications n
LEFT JOIN notification_seen ns 
    ON n.id = ns.notificationId AND ns.userId = $1
WHERE n.user1id = $1 OR n.user2id = $1
ORDER BY n.createdat DESC;
`
	rows, err := r.db.Query(query, userId)
	if err != nil {
		return domain.NotificationResponse{}, fmt.Errorf("failed to query notifications: %w", err)
	}
	defer rows.Close()

	var notifications []domain.Notificaion
	for rows.Next() {
		var notification domain.Notificaion
		if err := rows.Scan(
			&notification.ID,
			&notification.User1ID,
			&notification.User2ID,
			&notification.Message,
			&notification.CreatedAt,
			&notification.Seen,
		); err != nil {
			return domain.NotificationResponse{}, fmt.Errorf("failed to scan notification: %w", err)
		}
		notifications = append(notifications, notification)
	}
	if err := rows.Err(); err != nil {
		return domain.NotificationResponse{}, fmt.Errorf("error iterating over notifications: %w", err)
	}
	query = `SELECT COUNT(*) FROM waitlist WHERE userid = $1`
	var isWaiting bool
	err = r.db.QueryRow(query, userId).Scan(&isWaiting)
	if err != nil {
		return domain.NotificationResponse{}, fmt.Errorf("failed to check if user is waiting: %w", err)
	}
	return domain.NotificationResponse{Notifications: notifications, IsWaiting: isWaiting}, nil
}

func (r *UserRepoImpl) SeenNotification(userId int64) error {
	query := `INSERT INTO notification_seen (notificationId, userId)
SELECT id, $1 FROM notifications
WHERE user1id = $1 OR user2id = $1
ON CONFLICT (notificationId, userId) DO NOTHING;`
	_, err := r.db.Exec(query, userId)
	if err != nil {
		return fmt.Errorf("failed to mark notification as seen: %w", err)
	}
	return nil
}

func (r *UserRepoImpl) GeneratePair() (string, error) {
	// STEP 1: Fetch all pending pairs
	rows, err := r.db.Query(`SELECT id FROM pairs WHERE status = 'pending'`)
	if err != nil {
		return "", fmt.Errorf("failed to fetch pending pairs: %w", err)
	}
	defer rows.Close()

	var pairIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return "", err
		}
		pairIDs = append(pairIDs, id)
	}

	// STEP 2: Check participation
	for _, pairID := range pairIDs {
		rows2, err := r.db.Query(`
			SELECT userid, is_participating FROM pair_participation 
			WHERE pair_id = $1
		`, pairID)
		if err != nil {
			return "", fmt.Errorf("error checking participation for pair %s: %w", pairID, err)
		}
		defer rows2.Close()

		for rows2.Next() {
			var userID int64
			var isParticipating sql.NullBool
			if err := rows2.Scan(&userID, &isParticipating); err != nil {
				return "", err
			}

			if isParticipating.Valid && isParticipating.Bool {
				r.FillAttendance([]int64{userID})
				fmt.Printf("✅ User %d attended in pair %s\n", userID, pairID)
			} else {
				r.MissAttendance(userID)
				fmt.Printf("❌ User %d missed in pair %s\n", userID, pairID)
			}
		}
	}

	// STEP 3: Delete all old data (cleanup)
	_, err = r.db.Exec(`DELETE FROM pair_participation`)
	if err != nil {
		return "", fmt.Errorf("failed to clear pair_participation: %w", err)
	}

	_, err = r.db.Exec(`DELETE FROM pairs`)
	if err != nil {
		return "", fmt.Errorf("failed to clear pairs: %w", err)
	}
	_, err = r.db.Exec(`DELETE FROM waitlist`)
	if err != nil {
		return "", fmt.Errorf("failed to clear waitlist: %w", err)
	}
	_, err = r.db.Exec(`DELETE FROM notification_seen`)
	if err != nil {
		return "", fmt.Errorf("failed to clear notification_seen: %w", err)
	}
	_, err = r.db.Exec(`DELETE FROM notifications`)
	if err != nil {
		return "", fmt.Errorf("failed to clear notifications: %w", err)
	}

	// STEP 4: Generate new pairs
	message, err := services.GenerateDailyPairs(r.db, r.rtdbClient)
	if err != nil {
		return "", fmt.Errorf("failed to generate daily pairs: %w", err)
	}

	return message, nil
}
