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
	"strconv"
	"time"

	repo "lingo-backend/controllers/repository"
	domain "lingo-backend/domain"

	"github.com/cloudinary/cloudinary-go/v2"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"

	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

func ListenToBot(db *sql.DB) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
		return
	}
	var bot *tgbotapi.BotAPI
	for {
		bot, err = tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
		if err != nil {
			log.Println("⚠️ Telegram bot connection failed, retrying in 10s:", err)
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
				profilePhotoURL := getUserProfilePhoto(bot, int64(userID))
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
					log.Println("✅ User upserted in Firebase:", userID)
				}
			}
		}
	}
}

func generateOTP(length int) int64 {
	rand.Seed(time.Now().UnixNano())
	otpStr := ""
	for i := 0; i < length; i++ {
		digit := rand.Intn(10)
		otpStr += fmt.Sprint(digit)
	}
	otp, _ := strconv.ParseInt(otpStr, 10, 64)
	return otp
}

func getUserProfilePhoto(bot *tgbotapi.BotAPI, userID int64) string {
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

	// Step 1: Download from Telegram
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
	defer os.Remove(tempFile.Name()) // Delete after upload

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		log.Println("Error saving photo:", err)
		return ""
	}

	// Step 2: Upload to Cloudinary
	cld, _ := cloudinary.NewFromParams(os.Getenv("CLOUD_NAME"), os.Getenv("API_KEY"), os.Getenv("API_SECRET"))

	uploadResp, err := cld.Upload.Upload(context.Background(), tempFile.Name(), uploader.UploadParams{})
	if err != nil {
		log.Println("Cloudinary upload error:", err)
		return ""
	}

	log.Println("Public URL:", uploadResp.SecureURL)
	return uploadResp.SecureURL
}
