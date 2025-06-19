package repository

import (
	"database/sql"
	"lingo-backend/domain"
)

type PairRepositoryImpl struct {
	db *sql.DB
}

func NewPairRepository(db *sql.DB) *PairRepositoryImpl {
	return &PairRepositoryImpl{db: db}
}

func (s *PairRepositoryImpl) GetDailyPairs(userId int64) (domain.Pair, error) {
	var pair domain.Pair

	query := `
	SELECT 
		p.id, 
		p.user1id, p.user2id, p.user3id,
		p.username1, p.username2, p.username3,
		COALESCE(pp1.is_participating, false) AS user1participating,
		COALESCE(pp2.is_participating, false) AS user2participating,
		COALESCE(pp3.is_participating, false) AS user3participating,
		(p.user3id IS NOT NULL) AS specialgroup
	FROM pairs p
	LEFT JOIN pair_participation pp1 ON pp1.pair_id = p.id AND pp1.userid = p.user1id
	LEFT JOIN pair_participation pp2 ON pp2.pair_id = p.id AND pp2.userid = p.user2id
	LEFT JOIN pair_participation pp3 ON pp3.pair_id = p.id AND pp3.userid = p.user3id
	WHERE (p.user1id = $1 OR p.user2id = $1 OR p.user3id = $1)
	  AND p.date = CURRENT_DATE
	LIMIT 1;
	`

	err := s.db.QueryRow(query, userId).Scan(
		&pair.ID,
		&pair.User1ID, &pair.User2ID, &pair.User3ID,
		&pair.Username1, &pair.Username2, &pair.Username3,
		&pair.User1Participating, &pair.User2Participating, &pair.User3Participating,
		&pair.SpecialGroup,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return pair, nil // No pair found for today
		}
		return pair, err
	}

	return pair, nil
}

func (s *PairRepositoryImpl) UpdatePairParticipation(pairId int64, userId int64, participating bool) error {
	query := `
	INSERT INTO pair_participation (pair_id, userid, is_participating)
	VALUES ($1, $2, $3)
	ON CONFLICT (pair_id, userid) DO UPDATE SET is_participating = $3;
	`

	_, err := s.db.Exec(query, pairId, userId, participating)
	return err
}
