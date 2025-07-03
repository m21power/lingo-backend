package usecase

import (
	"lingo-backend/domain"
	util "lingo-backend/utils"
)

type UserUsecase struct {
	userRepo domain.UserRepository
}

func NewUserUsecase(userRepo domain.UserRepository) *UserUsecase {
	return &UserUsecase{
		userRepo: userRepo,
	}
}

func (u *UserUsecase) FillAttendance(userIds []int64) error {
	return u.userRepo.FillAttendance(userIds)
}

func (u *UserUsecase) MissAttendance(userId int64) error {
	return u.userRepo.MissAttendance(userId)
}

func (u *UserUsecase) PairUser(userId int64, username string, profileUrl string) (util.PairResponse, error) {
	return u.userRepo.PairUser(userId, username, profileUrl)
}

func (u *UserUsecase) GetNotifications(userId int64) ([]domain.Notificaion, error) {
	return u.userRepo.GetNotifications(userId)
}

func (u *UserUsecase) SeenNotification(userId int64) error {
	return u.userRepo.SeenNotification(userId)
}
