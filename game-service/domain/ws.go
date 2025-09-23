package domain

import (
	"sync"

	"github.com/gofiber/contrib/websocket"
	"github.com/google/uuid"
)

type WebSocketErrorMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"` // Opsiyonel: Hata kodu ekleyebilirsin
}
type Client struct {
	ID             uuid.UUID
	RoomID         uuid.UUID
	CurrentChannel uuid.UUID
	Send           chan []byte
	Conn           *websocket.Conn
	WriteLock      sync.Mutex
	Done           chan struct{}
}
