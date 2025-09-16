package handler

import (
	//"auth-service/internal/game" // game paketini import edin

	"context"
	"encoding/json"
	"fmt"
	"game-service/internal/api/game"
	"log"

	"github.com/gofiber/contrib/websocket"
)

type WebSocketListenRequest struct {
}

// WebSocketRequest, WebSocket üzerinden gelen mesajları temsil eder.
type WebSocketRequest struct {
	Type   string          `json:"type"` // Mesaj tipi (örn: "join_room", "chat_message", "draw_data")
	Data   json.RawMessage `json:"data"` // Mesajın içeriği
	RoomID string          `json:"roomId,omitempty"`
}

type KickRequest struct {
	PlayerID string `json:"playerID"`
}

type BanRequest struct {
	PlayerID string `json:"playerID"`
}

// WebSocketHandler, WebSocket bağlantılarını ve mesaj akışını yönetir.
type WebSocketHandler struct {
	roomManager *game.RoomManager
}

// NewWebSocketHandler, yeni bir WebSocketHandler örneği oluşturur.
func NewWebSocketHandler(rm *game.RoomManager) *WebSocketHandler {
	return &WebSocketHandler{
		roomManager: rm,
	}
}

// HandleWS, WebSocket bağlantısını kurar ve gelen mesajları işler.
func (h *WebSocketHandler) HandleWS(c *websocket.Conn, ctx context.Context, req *WebSocketListenRequest) {
	// Bu kısımda kullanıcı kimliği doğrulaması yapılmalıdır.
	// Örneğin, çerezden veya sorgu parametresinden token alıp auth servisi ile doğrulamak.
	// Şimdilik varsayılan bir kullanıcı kimliği kullanalım.
	playerID := c.Params("id") // URL'den oyuncu ID'sini al (örn: /ws/:id)

	// Yeni bir oyuncu nesnesi oluştur
	player := &game.Player{
		ID:     playerID,
		Conn:   c,
		Online: true,
	}

	// Bağlantı kapandığında temizleme işlemlerini yap
	defer func() {
		// Oyuncunun bulunduğu odayı bul
		room := h.roomManager.FindRoomByPlayerID(player.ID)
		if room != nil {
			room.RemovePlayer(player.ID)

			// Odadaki diğer oyunculara birinin ayrıldığını bildir
			byeMessage := map[string]string{
				"type": "player_left",
				"data": fmt.Sprintf("Oyuncu %s odadan ayrıldı.", player.ID),
			}
			jsonMsg, _ := json.Marshal(byeMessage)
			room.BroadcastMessage(websocket.TextMessage, jsonMsg)

			log.Printf("Oyuncu %s odadan ayrıldı.", player.ID)

			// Eğer odadaki son oyuncuysa odayı sil
			if len(room.Players) == 0 {
				h.roomManager.DeleteRoom(room.ID)
			}
		}
	}()

	// Gelen mesajları dinle
	for {
		mt, rawMsg, err := c.ReadMessage()
		if err != nil {
			log.Println("Okuma hatası:", err)
			break
		}

		var req WebSocketRequest
		if err := json.Unmarshal(rawMsg, &req); err != nil {
			log.Println("Mesajı JSON'a dönüştürme hatası:", err)
			continue
		}

		switch req.Type {
		case "create_room":
			// Oda oluşturma isteğini işler
			var roomData struct {
				Name       string `json:"name"`
				Mode       int    `json:"mode"`
				MaxPlayers int    `json:"maxPlayers"`
			}
			if err := json.Unmarshal(req.Data, &roomData); err != nil {
				log.Println("Oda oluşturma verisi hatası:", err)
				continue
			}
			newRoom := h.roomManager.CreateRoom(roomData.Name, roomData.Mode, roomData.MaxPlayers, player.ID)
			newRoom.AddPlayer(player)

			// Oyuncuya odaya katıldığını bildir
			response := map[string]interface{}{
				"type": "room_created",
				"data": newRoom,
			}
			jsonResponse, _ := json.Marshal(response)
			if err := c.WriteMessage(mt, jsonResponse); err != nil {
				log.Println("Yanıt gönderme hatası:", err)
			}

		case "join_room":
			// Odaya katılma isteğini işler
			room := h.roomManager.GetRoom(req.RoomID)
			if room == nil {
				errMsg := map[string]string{"type": "error", "message": "Oda bulunamadı."}
				jsonMsg, _ := json.Marshal(errMsg)
				c.WriteMessage(websocket.TextMessage, jsonMsg)
				continue
			}
			// BAN KONTROLÜ: Oyuncunun banlı olup olmadığını kontrol et
			if room.IsBanned(player.ID) {
				errMsg := map[string]string{"type": "error", "message": "Bu odaya girişin yasaklandı."}
				jsonMsg, _ := json.Marshal(errMsg)
				c.WriteMessage(websocket.TextMessage, jsonMsg)
				c.Close() // Bağlantıyı hemen kapat
				continue
			}
			room.AddPlayer(player)

			// Odaya katılan oyunculara haber ver
			joinMessage := map[string]string{"type": "player_joined", "data": player.ID}
			jsonMsg, _ := json.Marshal(joinMessage)
			room.BroadcastMessage(websocket.TextMessage, jsonMsg)

		case "chat_message":
			// Sohbet mesajlarını odaya yayınla
			room := h.roomManager.FindRoomByPlayerID(player.ID)
			if room != nil {
				// Mesajı tüm oyunculara yayınla
				room.BroadcastMessage(mt, rawMsg)
			}

		case "draw_data":
			// Çizim verilerini odaya yayınla
			room := h.roomManager.FindRoomByPlayerID(player.ID)
			if room != nil {
				// Çizim verisini diğer oyunculara gönder
				room.BroadcastMessage(mt, rawMsg)
			}

		case "kick_player":
			var banReq KickRequest
			if err := json.Unmarshal(req.Data, &banReq); err != nil {
				log.Println("Ban verisi hatası:", err)
				continue
			}

			// İsteği yapan oyuncunun kimliği
			requesterID := player.ID

			// Oyuncunun bulunduğu odayı bul
			room := h.roomManager.FindRoomByPlayerID(requesterID)
			if room == nil {
				errMsg := map[string]string{"type": "error", "message": "Oda bulunamadı veya yetkiniz yok."}
				jsonMsg, _ := json.Marshal(errMsg)
				c.WriteMessage(websocket.TextMessage, jsonMsg)
				continue
			}

			// YETKİLENDİRME KONTROLÜ
			// İsteği yapan oyuncunun oda sahibi olup olmadığını kontrol et.
			if room.CreatorID != requesterID {
				errMsg := map[string]string{"type": "error", "message": "Bu işlemi yapmaya yetkiniz yok."}
				jsonMsg, _ := json.Marshal(errMsg)
				c.WriteMessage(websocket.TextMessage, jsonMsg)
				continue
			}

			// Kick işlemi için hedef oyuncunun odada olup olmadığını kontrol et.
			// Kendini kick'lemeye çalışmasını da engelle.
			if requesterID == banReq.PlayerID {
				errMsg := map[string]string{"type": "error", "message": "Kendinizi odadan atamazsınız."}
				jsonMsg, _ := json.Marshal(errMsg)
				c.WriteMessage(websocket.TextMessage, jsonMsg)
				continue
			}

			// Kick işlemini gerçekleştir
			if err := room.BanPlayer(banReq.PlayerID); err != nil {
				log.Printf("Ban işlemi başarısız: %v", err)
				errMsg := map[string]string{"type": "error", "message": err.Error()}
				jsonMsg, _ := json.Marshal(errMsg)
				c.WriteMessage(websocket.TextMessage, jsonMsg)
				continue
			}

			// Odaya, bir oyuncunun banlandığına dair mesajı yayınla
			banMessage := map[string]string{
				"type": "player_banned",
				"data": fmt.Sprintf("Oyuncu %s odadan atıldı ve banlandı.", banReq.PlayerID),
			}
			jsonMsg, _ := json.Marshal(banMessage)
			room.BroadcastMessage(websocket.TextMessage, jsonMsg)
		case "unban_player":
			var unbanReq BanRequest
			if err := json.Unmarshal(req.Data, &unbanReq); err != nil {
				log.Println("Ban verisi hatası:", err)
				continue
			}

			// Yetki kontrolü (oda sahibi mi?)
			requesterID := player.ID
			room := h.roomManager.FindRoomByPlayerID(requesterID)
			if room == nil || room.CreatorID != requesterID {
				errMsg := map[string]string{"type": "error", "message": "Bu işlemi yapmaya yetkiniz yok."}
				jsonMsg, _ := json.Marshal(errMsg)
				c.WriteMessage(websocket.TextMessage, jsonMsg)
				continue
			}

			// Banı kaldır
			if err := room.UnbanPlayer(unbanReq.PlayerID); err != nil {
				errMsg := map[string]string{"type": "error", "message": err.Error()}
				jsonMsg, _ := json.Marshal(errMsg)
				c.WriteMessage(websocket.TextMessage, jsonMsg)
				continue
			}

			// Odaya, bir oyuncunun banının kaldırıldığına dair mesajı yayınla
			unbanMessage := map[string]string{
				"type": "player_unbanned",
				"data": fmt.Sprintf("Oyuncu %s banı kaldırıldı.", unbanReq.PlayerID),
			}
			jsonMsg, _ := json.Marshal(unbanMessage)
			room.BroadcastMessage(websocket.TextMessage, jsonMsg)
		}
	}
}
