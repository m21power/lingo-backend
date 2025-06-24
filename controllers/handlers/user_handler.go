package handlers

import (
	"encoding/json"
	"lingo-backend/usecase"
	util "lingo-backend/utils"
	"net/http"
)

type UserHandler struct {
	usecase usecase.UserUsecase
}

func NewUserHandler(usecase usecase.UserUsecase) *UserHandler {
	return &UserHandler{
		usecase: usecase,
	}
}
func (h *UserHandler) FillAttendance(w http.ResponseWriter, r *http.Request) {
	type UserIdsRequest struct {
		UserIds []int64 `json:"userIds"`
	}

	var userIds UserIdsRequest
	if err := json.NewDecoder(r.Body).Decode(&userIds); err != nil {
		util.WriteError(w, err, http.StatusBadRequest)
		return
	}
	if err := h.usecase.FillAttendance(userIds.UserIds); err != nil {
		util.WriteError(w, err, http.StatusInternalServerError)
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]string{"message": "Attendance filled successfully"})

}
