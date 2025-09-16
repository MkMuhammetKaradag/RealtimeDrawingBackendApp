package game

import (
	"sync"

	"github.com/gofiber/fiber/v2/log"
)

// RoomManager, tüm aktif oyun odalarını yönetir.
type RoomManager struct {
	Rooms map[string]*Room // Key: Room ID
	mu    sync.RWMutex
}

// NewRoomManager, yeni bir RoomManager örneği oluşturur.
func NewRoomManager() *RoomManager {
	return &RoomManager{
		Rooms: make(map[string]*Room),
	}
}

// CreateRoom, yeni bir oyun odası oluşturur ve yöneticiye ekler.
func (rm *RoomManager) CreateRoom(name string, mode, maxPlayers int) *Room {
	room := NewRoom(name, mode, maxPlayers)
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.Rooms[room.ID] = room
	log.Infof("Yeni oda oluşturuldu: %s (Mod: %d)", room.ID, mode)
	return room
}

// GetRoom, ID'ye göre bir odayı döndürür.
func (rm *RoomManager) GetRoom(roomID string) *Room {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.Rooms[roomID]
}

// DeleteRoom, bir odayı yöneticiden siler.
func (rm *RoomManager) DeleteRoom(roomID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	delete(rm.Rooms, roomID)
	log.Infof("Oda silindi: %s", roomID)
}

// FindRoomByPlayerID, bir oyuncunun bulunduğu odayı bulur.
func (rm *RoomManager) FindRoomByPlayerID(playerID string) *Room {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	for _, room := range rm.Rooms {
		room.mu.RLock()
		if _, exists := room.Players[playerID]; exists {
			room.mu.RUnlock()
			return room
		}
		room.mu.RUnlock()
	}
	return nil
}
