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
func (h *WebSocketRoomHandler) sendErrorAndClose(conn *websocket.Conn, msg string, code int) {
	errorMessage := domain.WebSocketErrorMessage{
		Type:    "error",
		Message: msg,
		Code:    code,
	}
	if err := conn.WriteJSON(errorMessage); err != nil {
		fmt.Printf("Failed to send error message to client: %v\n", err)
	}
	conn.Close()
}

// HandleWS metodunuzu güncelleyin
func (h *WebSocketRoomHandler) HandleWS(c *websocket.Conn, ctx context.Context, req *WebSocketRoomRequest) {
	userID := c.Headers("X-User-Id")

	currentUserID, err := uuid.Parse(userID)
	if err != nil {
		h.sendErrorAndClose(c, fmt.Sprintf("Failed to parse user ID: %v", err), fiber.StatusBadRequest)
		return
	}

	roomID, err := uuid.Parse(c.Params("room_id"))
	if err != nil {
		h.sendErrorAndClose(c, fmt.Sprintf("Failed to parse room ID: %v", err), fiber.StatusBadRequest)
		return
	}

	h.usecase.Execute(c, ctx, roomID, currentUserID)
}
