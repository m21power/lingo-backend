package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"lingo-backend/domain"
	"time"

	"cloud.google.com/go/firestore"
)

type OtpRepositoryImpl struct {
	db        *sql.DB
	firestore *firestore.Client
}

func NewOtpRepository(db *sql.DB, firestore *firestore.Client) *OtpRepositoryImpl {
	return &OtpRepositoryImpl{
		db:        db,
		firestore: firestore,
	}
}

func (r *OtpRepositoryImpl) SaveOtp(otp domain.Otp) error {
	query := `INSERT INTO otp (userid, otp, username, createdat) VALUES ($1, $2, $3, NOW())`
	_, err := r.db.Exec(query, otp.UserID, otp.Otp, otp.Username)
	if err != nil {
		return err
	}
	return nil
}
func (r *OtpRepositoryImpl) CheckOtp(username string, otp int64) (*domain.User, error) {
	query := `SELECT createdat FROM otp WHERE LOWER(username) = LOWER($1) AND otp = $2`
	var createdAt sql.NullTime

	err := r.db.QueryRow(query, username, otp).Scan(&createdAt)
	if err == sql.ErrNoRows {
		return nil, errors.New("Invalid OTP")
	}
	if err != nil {
		return nil, err
	}

	query = `SELECT userid FROM otp WHERE otp=$1 AND LOWER(username)=LOWER($2)`
	var userID int64
	err = r.db.QueryRow(query, otp, username).Scan(&userID)
	if err != nil {
		return nil, err
	}
	user, err := r.firestore.Collection("users").Doc(fmt.Sprint(userID)).Get(context.Background())
	if err != nil {
		return nil, err
	}
	println("user", user.Data())
	expiration := createdAt.Time.Add(30 * time.Minute)
	if !createdAt.Valid || expiration.Before(time.Now()) {
		_, _ = r.db.Exec(`DELETE FROM otp WHERE LOWER(username) = LOWER($1) AND otp = $2`, username, otp)
		return nil, errors.New("OTP has expired. Please request a new one.")
	}

	_, err = r.db.Exec(`DELETE FROM otp WHERE LOWER(username) = LOWER($1) AND otp = $2`, username, otp)
	if err != nil {
		return nil, err
	}

	return &domain.User{
		ID:         safeInt64(user.Data()["userId"]),
		Username:   safeString(user.Data()["username"]),
		PhotoUrl:   safeString(user.Data()["profileUrl"]),
		MissCount:  safeInt64(user.Data()["missCount"]),
		Attendance: safeInt64(user.Data()["attendance"]),
	}, nil

}
func safeInt64(value interface{}) int64 {
	if value == nil {
		return 0
	}
	if v, ok := value.(int64); ok {
		return v
	}
	if v, ok := value.(float64); ok {
		return int64(v)
	}
	return 0
}

func safeString(value interface{}) string {
	if value == nil {
		return ""
	}
	if v, ok := value.(string); ok {
		return v
	}
	return ""
}

func InsertOtp(db *sql.DB, otp domain.Otp) error {
	query := `
	INSERT INTO otp (userid, otp, username, createdat)
	VALUES ($1, $2, $3, NOW())
	ON CONFLICT (userid, username)
	DO UPDATE SET
		otp = EXCLUDED.otp,
		createdat = NOW()
`

	_, err := db.Exec(query, otp.UserID, otp.Otp, otp.Username)
	if err != nil {
		return err
	}
	return nil
}
