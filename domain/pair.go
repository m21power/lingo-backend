package domain

type Pair struct {
	ID                 int64  `json:"id" db:"id"`
	User1ID            int64  `json:"user1Id" db:"user1id"`
	User2ID            int64  `json:"user2Id" db:"user2id"`
	User3ID            int64  `json:"user3Id" db:"user3id"`
	Username1          string `json:"username1" db:"username1"`
	Username2          string `json:"username2" db:"username2"`
	Username3          string `json:"username3" db:"username3"`
	User1Participating bool   `json:"user1Participating" db:"user1participating"`
	User2Participating bool   `json:"user2Participating" db:"user2participating"`
	User3Participating bool   `json:"user3Participating" db:"user3participating"`
	SpecialGroup       bool   `json:"specialGroup" db:"specialgroup"`
}

type PairRepository interface {
	GetDailyPairs(userId int64) (Pair, error)
	UpdatePairParticipation(pairId int64, userId int64, participating bool) error
}
