package domain

import util "lingo-backend/utils"

type Notificaion struct {
	ID        string `json:"id" db:"id"`
	User1ID   int64  `json:"user1Id" db:"user1id"`
	User2ID   int64  `json:"user2Id" db:"user2id"`
	Message   string `json:"message" db:"message"`
	Seen      bool   `json:"seen" db:"seen"`
	CreatedAt string `json:"createdAt" db:"createdat"`
}
type NotificationResponse struct {
	Notifications []Notificaion `json:"notifications"`
	IsWaiting     bool          `json:"isWaiting"`
}

type UserRepository interface {
	FillAttendance(userIds []int64) error
	MissAttendance(userId int64) error
	PairUser(userId int64, username string, profileUrl string) (util.PairResponse, error)
	GetNotifications(userId int64) (NotificationResponse, error)
	SeenNotification(userId int64) error
	GeneratePair() (string, error)
}
