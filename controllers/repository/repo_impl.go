package repository

import (
	"database/sql"
	"errors"
	"lingo-backend/domain"
	"time"
)

type OtpRepositoryImpl struct {
	db *sql.DB
}

func NewOtpRepository(db *sql.DB) *OtpRepositoryImpl {
	return &OtpRepositoryImpl{
		db: db,
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
func (r *OtpRepositoryImpl) CheckOtp(username string, otp int64) (bool, error) {
	query := `SELECT createdat FROM otp WHERE LOWER(username) = LOWER($1) AND otp = $2`
	var createdAt sql.NullTime

	err := r.db.QueryRow(query, username, otp).Scan(&createdAt)
	if err == sql.ErrNoRows {
		return false, errors.New("Invalid OTP")
	}
	if err != nil {
		return false, err
	}

	expiration := createdAt.Time.Add(30 * time.Minute)
	if !createdAt.Valid || expiration.Before(time.Now()) {
		_, _ = r.db.Exec(`DELETE FROM otp WHERE LOWER(username) = LOWER($1) AND otp = $2`, username, otp)
		return false, errors.New("OTP has expired. Please request a new one.")
	}

	_, err = r.db.Exec(`DELETE FROM otp WHERE LOWER(username) = LOWER($1) AND otp = $2`, username, otp)
	if err != nil {
		return false, err
	}

	return true, nil
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
