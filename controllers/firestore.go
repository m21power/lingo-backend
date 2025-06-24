package controllers

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type User struct {
	UserID            int64     `firestore:"userId"`
	Username          string    `firestore:"username"`
	ProfileURL        string    `firestore:"profileUrl"`
	MissCount         int64     `firestore:"missCount"`
	Attendance        int64     `firestore:"attendance"`
	ParticipatedCount int64     `firestore:"participatedCount"`
	CreatedAt         time.Time `firestore:"createdAt"`
}

func UpsertUserToFirebase(user User) error {
	ctx := context.Background()
	opt := option.WithCredentialsFile("lingo-firestore.json")

	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return err
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	docRef := client.Collection("users").Doc(fmt.Sprint(user.UserID))

	// Use a transaction to check existence and update or create accordingly
	err = client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		docSnap, err := tx.Get(docRef)
		if err != nil && grpc.Code(err) != codes.NotFound {
			return err
		}

		if !docSnap.Exists() {
			// User does not exist, create with createdAt = now
			user.CreatedAt = time.Now()
			user.MissCount = 0
			user.Attendance = 0
			user.ParticipatedCount = 0
			return tx.Set(docRef, user)
		}

		// User exists, update only username and profileUrl (keep createdAt)
		return tx.Update(docRef, []firestore.Update{
			{Path: "username", Value: user.Username},
			{Path: "profileUrl", Value: user.ProfileURL},
		})
	})

	if err != nil {
		return err
	}

	log.Println("✅ User upserted in Firebase:", user.UserID)
	return nil
}
