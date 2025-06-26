package handlers

import (
	"bufio"
	"bytes"
	"encoding/json"
	"lingo-backend/usecase"
	util "lingo-backend/utils"
	"log"
	"net/http"
	"os"

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

func (h *UserHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}
		go handleStreamToOpenRouter(conn, string(msg))
	}
}

func handleStreamToOpenRouter(conn *websocket.Conn, question string) {
	apiKey := os.Getenv("OPEN_ROUTER_API_KEY")
	url := "https://openrouter.ai/api/v1/chat/completions"

	payload := map[string]interface{}{
		"model": "openai/gpt-4o",
		"messages": []map[string]string{
			{"role": "user", "content": question},
		},
		"stream": true,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte("❌ Failed to connect to OpenRouter"))
		return
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			break
		}

		if bytes.HasPrefix(line, []byte("data: ")) {
			data := bytes.TrimPrefix(line, []byte("data: "))
			if bytes.Contains(data, []byte("[DONE]")) {
				break
			}

			var jsonData map[string]interface{}
			if err := json.Unmarshal(data, &jsonData); err == nil {
				choices := jsonData["choices"].([]interface{})
				delta := choices[0].(map[string]interface{})["delta"].(map[string]interface{})
				if content, ok := delta["content"].(string); ok {
					conn.WriteMessage(websocket.TextMessage, []byte(content))
				}
			}
		}
	}
}
