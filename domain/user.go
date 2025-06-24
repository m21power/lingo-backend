package domain

type UserRepository interface {
	FillAttendance(userIds []int64) error
	MissAttendance(userId int64) error
}
