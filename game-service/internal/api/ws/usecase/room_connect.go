package wsUsecase

import (
	"context"
	"fmt"
	"game-service/domain"
	"game-service/internal/api/ws/hub"

	"github.com/gofiber/contrib/websocket"
	"github.com/google/uuid"
)

type RoomManagerUseCase interface {
	Execute(c *websocket.Conn, ctx context.Context, roomID, currentUserID uuid.UUID)
}
type roomManagerUseCase struct {
	hub        Hub
	repository PostgresRepository
}

func NewRoomManagerUseCase(hub Hub, repository PostgresRepository) RoomManagerUseCase {
	return &roomManagerUseCase{
		hub:        hub,
		repository: repository,
	}
}

func (u *roomManagerUseCase) Execute(c *websocket.Conn, ctx context.Context, roomID, currentUserID uuid.UUID) {

	sendErrorToClient := func(conn *websocket.Conn, msg string) {
		errorMessage := domain.WebSocketErrorMessage{
			Type:    "error",
			Message: msg,
		}
		if err := conn.WriteJSON(errorMessage); err != nil {
			fmt.Printf("Failed to send error message to client: %v\n", err)
		}
	}

	// 1. Oda üyeliği kontrolü
	isMember, err := u.repository.IsMemberRoom(ctx, roomID, currentUserID)
	if err != nil || !isMember {
		errMsg := fmt.Sprintf("Authorization error: %v", err)
		sendErrorToClient(c, errMsg)
		fmt.Printf("User %s is not a member of room %s: %v\n", currentUserID, roomID, err)
		c.Close() // ❌ Bağlantıyı kapat
		return
	}

	// 2. Oyun durumu kontrolü
	if u.hub.IsGameActive(roomID) {
		game := u.hub.GetActiveGame(roomID)
		if game == nil {
			sendErrorToClient(c, "Oyun durumu alınamadı. Lütfen daha sonra tekrar deneyin.")
			c.Close()
			return
		}

		// 🔍 Kullanıcı oyuncu listesinde mi kontrol et
		if !u.hub.IsPlayerInActiveGame(roomID, currentUserID) {
			sendErrorToClient(c, "Bu odada zaten bir oyun devam ediyor. Oyun bittikten sonra tekrar deneyin.")
			c.Close()
			return
		}

		// ✅ Oyuncu zaten oyundaysa, yeniden bağlanmasına izin ver (reconnect durumu)
		fmt.Printf("Player %s reconnecting to active game in room %s\n", currentUserID, roomID)
		u.sendGameStateOnConnect(c, game)
		u.hub.BroadcastMessage(roomID, &hub.Message{
			Type: "player_reconnected",
			Content: map[string]interface{}{
				"room_id": roomID,
				"user_id": currentUserID,
				"message": "Oyuncu tekrar bağlandı",
			},
		})
	} else {
		// Oyun aktif değil, bekleme durumunu gönder
		u.sendWaitingStateOnConnect(c, roomID)
	}

	// 3. Client'ı Hub'a Kaydet
	client := &domain.Client{
		ID:     currentUserID,
		Conn:   c,
		RoomID: roomID,
		Send:   make(chan []byte, 256),
	}
	fmt.Printf("Registering client %s to room %s\n", currentUserID, roomID)
	u.hub.RegisterClient(client)

	select {}
}
func (u *roomManagerUseCase) isPlayerInGame(game *hub.Game, userID uuid.UUID) bool {
	for _, player := range game.Players {
		if player.UserID == userID {
			return true
		}
	}
	return false
}
func (u *roomManagerUseCase) sendGameStateOnConnect(conn *websocket.Conn, game *hub.Game) {
	// Game nesnesini JSON'a dönüştürerek sadece bu yeni bağlanan istemciye gönder.
	// Bu, istemciye oyunun başladığını ve mevcut durumunu bildirir.

	// Örnek: Basit bir mesaj tipi gönderelim
	type GameStatusMessage struct {
		Type     string    `json:"type"`
		State    string    `json:"state"`
		GameData *hub.Game `json:"game_data,omitempty"`
	}

	// Güvenlik: ModeData'daki gizli bilgileri (örneğin DrawingGameData'daki CurrentWord) temizlemeyi unutmayın!

	msg := GameStatusMessage{
		Type:     "game_status",
		State:    game.State,
		GameData: game, // Bu kısımda hassas verileri temizlemek önemlidir.
	}

	if err := conn.WriteJSON(msg); err != nil {
		fmt.Printf("Failed to send game status to client: %v\n", err)
	}
}

func (u *roomManagerUseCase) sendWaitingStateOnConnect(conn *websocket.Conn, roomID uuid.UUID) {
	// Oyunun bekleme (waiting) durumunda olduğunu bildiren mesaj.
	type WaitingMessage struct {
		Type    string    `json:"type"`
		RoomID  uuid.UUID `json:"room_id"`
		Message string    `json:"message"`
	}

	msg := WaitingMessage{
		Type:    "room_status",
		RoomID:  roomID,
		Message: "Oda hazır, diğer oyuncular bekleniyor.",
	}

	if err := conn.WriteJSON(msg); err != nil {
		fmt.Printf("Failed to send waiting status to client: %v\n", err)
	}
}
