package domain

type Otp struct {
	ID        int64  `json:"id" db:"id"`
	UserID    int64  `json:"userId" db:"userid"`
	Otp       int64  `json:"otp" db:"otp"`
	Username  string `json:"username" db:"username"`
	CreatedAt string `json:"createdAt" db:"createdat"`
}

type OtpRepository interface {
	SaveOtp(otp Otp) error
	CheckOtp(username string, otp int64) (bool, error)
}
