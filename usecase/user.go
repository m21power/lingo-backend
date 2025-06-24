package usecase

import (
	"lingo-backend/domain"
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
