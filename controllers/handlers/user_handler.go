package handlers

import (
	"encoding/json"
	"lingo-backend/domain"
	"lingo-backend/usecase"
	util "lingo-backend/utils"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type UserHandler struct {
	usecase usecase.UserUsecase
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // allow any origin (adjust in production)
	},
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
func (h *UserHandler) PairUser(w http.ResponseWriter, r *http.Request) {
	type PairUserRequest struct {
		UserId     int64  `json:"userId"`
		Username   string `json:"username"`
		ProfileUrl string `json:"profileUrl"`
	}

	var pairUserReq PairUserRequest
	if err := json.NewDecoder(r.Body).Decode(&pairUserReq); err != nil {
		util.WriteError(w, err, http.StatusBadRequest)
		return
	}
	resp, err := h.usecase.PairUser(pairUserReq.UserId, pairUserReq.Username, pairUserReq.ProfileUrl)
	if err != nil {
		util.WriteError(w, err, http.StatusInternalServerError)
		return
	}
	if resp.Wait {
		util.WriteJSON(w, http.StatusOK, map[string]bool{"wait": true})
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]bool{"wait": false})
}
func (h *UserHandler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	value := vars["userId"]
	println("value:", value)
	userId, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		util.WriteError(w, err, http.StatusBadRequest)
		return
	}
	notifications, err := h.usecase.GetNotifications(userId)
	if err != nil {
		util.WriteError(w, err, http.StatusInternalServerError)
		return
	}

	util.WriteJSON(w, http.StatusOK, map[string]domain.NotificationResponse{
		"notifications": notifications,
	})
}

func (h *UserHandler) SeenNotification(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	value := vars["userId"]
	userId, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		util.WriteError(w, err, http.StatusBadRequest)
		return
	}
	err = h.usecase.SeenNotification(userId)
	if err != nil {
		util.WriteError(w, err, http.StatusInternalServerError)
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]string{"message": "Notifications marked as seen"})
}
