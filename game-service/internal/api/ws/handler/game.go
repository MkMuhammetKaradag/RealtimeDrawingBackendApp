package wsHandler

import (
	//"auth-service/internal/game" // game paketini import edin

	"context"

	"fmt"
	"game-service/domain"
	wsUsecase "game-service/internal/api/ws/usecase"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// WebSocketRoomHandler, WebSocket bağlantılarını ve mesaj akışını yönetir.
type WebSocketRoomHandler struct {
	usecase wsUsecase.RoomManagerUseCase
}
type WebSocketRoomRequest struct {
}

// NewWebSocketRoomHandler, yeni bir WebSocketRoomHandler örneği oluşturur.
func NewWebSocketRoomHandler(uscase wsUsecase.RoomManagerUseCase) *WebSocketRoomHandler {
	return &WebSocketRoomHandler{
		usecase: uscase,
	}
}

// HandleWS, WebSocket bağlantısını kurar ve gelen mesajları işler.
func (h *WebSocketRoomHandler) HandleWS(c *websocket.Conn, ctx context.Context, req *WebSocketRoomRequest) {

	sendErrorToClient := func(conn *websocket.Conn, msg string, code int) {
		errorMessage := domain.WebSocketErrorMessage{
			Type:    "error",
			Message: msg,
			Code:    code,
		}
		if err := conn.WriteJSON(errorMessage); err != nil {
			fmt.Printf("Failed to send error message to client: %v\n", err)
		}
	}
	userID := c.Headers("X-User-Id")
	// fmt.Println("session:", c)

	if userID == "" {
		sendErrorToClient(c, domain.ErrUnauthorized.Error(), fiber.StatusUnauthorized) // Hata mesajını istemciye gönder
		return

	}

	currentUserID, err := uuid.Parse(userID)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to parse user ID: %v", err)
		sendErrorToClient(c, errMsg, 0) // Hata mesajını istemciye gönder
		fmt.Errorf(errMsg)              // Sunucu tarafında da logla
		return
	}

	if err != nil {
		errMsg := fmt.Sprintf("Authentication failed: %v", err)
		sendErrorToClient(c, errMsg, 0) // Hata mesajını istemciye gönder         // Sunucu tarafında da logla
		return
	}
	roomID, err := uuid.Parse(c.Params("room_id"))
	if err != nil {
		errMsg := fmt.Sprintf("Failed to parse room ID: %v", err)
		sendErrorToClient(c, errMsg, 0)
		return
	}

	h.usecase.Execute(c, ctx, roomID, currentUserID)
}
