package handlers

import (
	"encoding/json"
	"lingo-backend/domain"
	usecase "lingo-backend/usecase"
	util "lingo-backend/utils"
	"net/http"
)

type OtpHandler struct {
	usecase usecase.OtpUsecase
}

func NewOtpHandler(otpUsecase usecase.OtpUsecase) *OtpHandler {
	return &OtpHandler{
		usecase: otpUsecase,
	}
}
func (h *OtpHandler) SaveOtp(otp domain.Otp) error {
	// we don't need handler for saving
	return nil
}
func (h *OtpHandler) CheckOtp(w http.ResponseWriter, r *http.Request) {
	var payload domain.Otp
	err := json.NewDecoder(r.Body).Decode(&payload)
	// println("Received payload:", payload)
	println("Received payload username:", payload.Username)
	println("Received payload otp:", payload.Otp)
	if err != nil {
		util.WriteError(w, err, http.StatusBadRequest)
		return
	}
	result, err := h.usecase.CheckOtp(payload.Username, payload.Otp)
	if err != nil {
		util.WriteError(w, err, http.StatusBadRequest)
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]*domain.User{"user": result})

}
