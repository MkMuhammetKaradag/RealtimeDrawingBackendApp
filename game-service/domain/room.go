package domain

import (
	"time"

	"github.com/google/uuid"
)

type Room struct {
	ID             uuid.UUID `json:"id"`
	RoomName       string    `json:"room_name"`
	CreatorID      uuid.UUID `json:"creator_id"`
	MaxPlayers     int       `json:"max_players"`
	CurrentPlayers int       `json:"current_players"`
	Status         string    `json:"status"`
	GameModeID     int       `json:"game_mode_id"`
	IsPrivate      bool      `json:"is_private"`
	RoomCode       string    `json:"room_code,omitempty"` // Gizli olabileceğinden omitempty ekledik
	CreatedAt      time.Time `json:"created_at"`
	StartedAt      time.Time `json:"started_at,omitempty"`
	FinishedAt     time.Time `json:"finished_at,omitempty"`
}
