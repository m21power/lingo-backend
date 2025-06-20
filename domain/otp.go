package domain

type Otp struct {
	ID        int64  `json:"id" db:"id"`
	UserID    int64  `json:"userId" db:"userid"`
	Otp       int64  `json:"otp" db:"otp"`
	Username  string `json:"username" db:"username"`
	CreatedAt string `json:"createdAt" db:"createdat"`
}

type User struct {
	ID             int64   `json:"id" db:"id"`
	Username       string  `json:"username" db:"username"`
	PhotoUrl       string  `json:"photoUrl" db:"photoUrl"`
	MissCount      int64   `json:"missCount" db:"missCount"`
	Attendance     int64   `json:"attendance" db:"attendance"`
	MissPercentage float64 `json:"missPercentage" db:"missPercentage"`
	CreatedAt      string  `json:"createdAt" db:"createdat"`
}

type OtpRepository interface {
	SaveOtp(otp Otp) error
	CheckOtp(username string, otp int64) (*User, error)
}
