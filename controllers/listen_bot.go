package controllers

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	repo "lingo-backend/controllers/repository"
	domain "lingo-backend/domain"

	"cloud.google.com/go/firestore"
	"github.com/cloudinary/cloudinary-go/v2"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

func ListenToBot(db *sql.DB, firestoreClient *firestore.Client) {
	// err := godotenv.Load()
	// if err != nil {
	// 	log.Fatal("Error loading .env file")
	// 	return
	// }
	var bot *tgbotapi.BotAPI
	var err error
	for {
		bot, err = tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
		if err != nil {
			log.Println(" Telegram bot connection failed, retrying in 10s:", err)
			time.Sleep(10 * time.Second)
			continue
		}
		break
	}

	bot.Debug = true // For logging

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil && update.Message.IsCommand() {
			if update.Message.Command() == "start" {
				user := update.Message.From
				chatID := update.Message.Chat.ID

				username := user.UserName
				userID := user.ID

				// Generate random OTP
				otp := generateOTP(4)
				var payload = domain.Otp{
					UserID:   userID,
					Otp:      otp,
					Username: username,
				}
				repo.InsertOtp(db, payload)

				msg := tgbotapi.NewMessage(chatID, "Hi @"+username+" 👋\nYour OTP is: "+fmt.Sprint(otp))
				bot.Send(msg)

				// Profile picture URL – requires extra call
				profilePhotoURL := getUserProfilePhoto(bot, int64(userID), firestoreClient)
				log.Println("User profile photo URL:", profilePhotoURL)
				var userProfile = User{
					UserID:     userID,
					Username:   username,
					ProfileURL: profilePhotoURL,
					CreatedAt:  time.Now(),
				}
				err := UpsertUserToFirebase(userProfile)
				if err != nil {
					log.Println("Error upserting user to Firebase:", err)
				} else {
					log.Println(" User upserted in Firebase:", userID)
				}
			}
		}
	}
}

func generateOTP(length int) int64 {
	rand.Seed(time.Now().UnixNano())
	min := int64(1)
	for i := 1; i < length; i++ {
		min *= 10
	}
	max := min*10 - 1
	return rand.Int63n(max-min+1) + min
}

func getUserProfilePhoto(bot *tgbotapi.BotAPI, userID int64, firestoreClient *firestore.Client) string {
	ctx := context.Background()

	// Step 0: Cloudinary setup
	cld, _ := cloudinary.NewFromParams(os.Getenv("CLOUD_NAME"), os.Getenv("API_KEY"), os.Getenv("API_SECRET"))

	// Step 1: Get current profile URL from Firestore
	userDoc := firestoreClient.Collection("users").Doc(fmt.Sprint(userID))
	docSnap, err := userDoc.Get(ctx)
	if err == nil && docSnap.Exists() {
		if oldUrl, ok := docSnap.Data()["profileUrl"].(string); ok && oldUrl != "" {
			// Extract public_id from the Cloudinary URL
			publicID := getPublicIDFromURL(oldUrl)
			if publicID != "" {
				_, err := cld.Upload.Destroy(ctx, uploader.DestroyParams{PublicID: publicID})
				if err != nil {
					log.Println("Cloudinary delete error:", err)
				}
				log.Println("Cloudinary delete success:", publicID)
			}
		}
	}

	// get telegram profile photo
	photos, err := bot.GetUserProfilePhotos(tgbotapi.UserProfilePhotosConfig{
		UserID: userID,
		Limit:  1,
	})
	if err != nil || photos.TotalCount == 0 {
		log.Println("Error getting photo or no photo found:", err)
		return ""
	}

	fileID := photos.Photos[0][0].FileID
	file, _ := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})

	downloadURL := "https://api.telegram.org/file/bot" + bot.Token + "/" + file.FilePath
	resp, err := http.Get(downloadURL)
	if err != nil {
		log.Println("Error downloading photo:", err)
		return ""
	}
	defer resp.Body.Close()

	tempFile, err := os.CreateTemp("", "profile_*.jpg")
	if err != nil {
		log.Println("Temp file error:", err)
		return ""
	}
	defer os.Remove(tempFile.Name())

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		log.Println("Error saving photo:", err)
		return ""
	}

	// upload new photo to cloudinary
	uploadResp, err := cld.Upload.Upload(ctx, tempFile.Name(), uploader.UploadParams{})
	if err != nil {
		log.Println("Cloudinary upload error:", err)
		return ""
	}

	// update firestore with new photo URL
	_, err = userDoc.Update(ctx, []firestore.Update{
		{Path: "profileUrl", Value: uploadResp.SecureURL},
	})
	if err != nil {
		log.Println("Error updating Firestore:", err)
	}

	log.Println("Public URL:", uploadResp.SecureURL)
	return uploadResp.SecureURL
}
func getPublicIDFromURL(url string) string {
	// Split the URL by "/" and get the last part
	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return ""
	}
	// Get the filename: profile_1314364420_bikyig.jpg
	filename := parts[len(parts)-1]

	// Remove the file extension (.jpg, .png, etc.)
	publicID := strings.TrimSuffix(filename, filepath.Ext(filename))
	return publicID
}
