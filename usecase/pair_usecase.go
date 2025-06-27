package usecase

import domain "lingo-backend/domain"

type PairUsecase struct {
	repository domain.PairRepository
}

func NewPairUsecase(repository domain.PairRepository) *PairUsecase {
	return &PairUsecase{
		repository: repository,
	}
}

func (u *PairUsecase) GetDailyPairs(userId int64) (domain.Pair, error) {
	return u.repository.GetDailyPairs(userId)
}
func (u *PairUsecase) UpdatePairParticipation(pairId string, userId int64, participating bool) error {
	return u.repository.UpdatePairParticipation(pairId, userId, participating)
}
