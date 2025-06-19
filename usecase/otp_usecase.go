package usecase

import domain "lingo-backend/domain"

type OtpUsecase struct {
	otpRepo domain.OtpRepository
}

func NewOtpUsecase(otpRepo domain.OtpRepository) *OtpUsecase {
	return &OtpUsecase{
		otpRepo: otpRepo,
	}
}
func (u *OtpUsecase) SaveOtp(otp domain.Otp) error {
	return u.otpRepo.SaveOtp(otp)
}
func (u *OtpUsecase) CheckOtp(username string, otp int64) (bool, error) {
	return u.otpRepo.CheckOtp(username, otp)
}
