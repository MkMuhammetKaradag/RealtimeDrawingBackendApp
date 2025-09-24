package wsUsecase

import (
	"context"
	"fmt"
	"game-service/domain"

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

	isMember, err := u.repository.IsMemberRoom(ctx, roomID, currentUserID)
	if err != nil || !isMember {
		errMsg := fmt.Sprintf("Authorization error: %v", err)
		sendErrorToClient(c, errMsg) // Hata mesajını istemciye gönder
		fmt.Errorf("is member :", err)
		return
	}

	client := &domain.Client{
		ID:     currentUserID,
		Conn:   c,
		RoomID: roomID,
		Send:   make(chan []byte, 256),
	}
	fmt.Println(client)
	u.hub.RegisterClient(client)
	select {}
	// defer func() {
	// 	u.hub.UnregisterClient(client)
	// }()

	// // Ping/Pong timeout ayarları
	// pingInterval := time.Second * 20
	// pongTimeout := time.Second * 50

	// // Ping ticker başlat
	// ticker := time.NewTicker(pingInterval)
	// defer ticker.Stop()

	// // Pong handler
	// c.SetPongHandler(func(string) error {
	// 	fmt.Println("Pong received") // Debug için
	// 	c.SetReadDeadline(time.Now().Add(pongTimeout))
	// 	return nil
	// })

	// // İlk read deadline
	// c.SetReadDeadline(time.Now().Add(pongTimeout))

	// // Goroutine for ping messages
	// go func() {
	// 	for {
	// 		select {
	// 		case <-ticker.C:
	// 			fmt.Println("Sending ping") // Debug için
	// 			if err := c.WriteMessage(websocket.PingMessage, nil); err != nil {
	// 				fmt.Printf("Ping send error: %v\n", err)
	// 				return
	// 			}
	// 		case <-ctx.Done():
	// 			return
	// 		}
	// 	}
	// }()

	// for {
	// 	_, _, err := c.ReadMessage()
	// 	if err != nil {
	// 		// Connection kapatıldı, defer çalışacak
	// 		fmt.Printf("ReadMessage error: %v\n", err) // Debug için
	// 		break
	// 	}
	// }
}
