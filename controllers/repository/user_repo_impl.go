package repository

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/lib/pq"
)

type UserRepoImpl struct {
	db        *sql.DB
	firestore *firestore.Client
}

func NewUserRepo(db *sql.DB, firestore *firestore.Client) *UserRepoImpl {
	return &UserRepoImpl{
		db:        db,
		firestore: firestore,
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

	// Get today's date string in YYYY-MM-DD format
	today := time.Now().Format("2006-01-02")

	for _, userId := range userIds {
		docID := strconv.FormatInt(userId, 10)

		_, err := r.firestore.Collection("users").Doc(docID).Update(ctx, []firestore.Update{
			{Path: "attendance", Value: firestore.Increment(1)},
		})
		if err != nil {
			return err
		}

		consistencyDoc := r.firestore.Collection("consistency").Doc(docID).Collection(today).Doc("daily_score")

		// Use Set with Merge to create or update, so only one doc per day
		_, err = consistencyDoc.Set(ctx, map[string]interface{}{
			"score": 1,
		}, firestore.MergeAll)
		if err != nil {
			return err
		}
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

	// Get today's date string in YYYY-MM-DD format
	today := time.Now().Format("2006-01-02")
	consistencyDoc := r.firestore.Collection("consistency").Doc(docID).Collection(today).Doc("daily_score")

	// Use Set with Merge to create or update, so only one doc per day
	_, err = consistencyDoc.Set(ctx, map[string]interface{}{
		"score": 1,
	}, firestore.MergeAll)
	if err != nil {
		return err
	}

	return nil
}
