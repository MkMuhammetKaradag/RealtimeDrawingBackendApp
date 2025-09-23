package hub

import (
	"context"
	"encoding/json"
	"log"

	"github.com/redis/go-redis/v9"
)

type RoomManagerHub struct {
	redisClient *redis.Client
	parentHub   *Hub
}

func NewRoomManagerHub(redisClient *redis.Client, parent *Hub) *RoomManagerHub {
	return &RoomManagerHub{
		redisClient: redisClient,
		parentHub:   parent,
	}
}

func (sh *RoomManagerHub) Run(ctx context.Context) {

	go sh.RoomManager(ctx)

}

func (sh *RoomManagerHub) RoomManager(ctx context.Context) {
	pubsub := sh.redisClient.Subscribe(ctx, "room_manager")
	defer pubsub.Close()

	// Kanal dinleme döngüsü
	for {
		select {
		case <-ctx.Done():
			// Bağlam iptal edildiğinde çık
			return

		default:
			// Redis'ten mesaj al
			msg, err := pubsub.ReceiveMessage(ctx)
			if err != nil {
				log.Println("Redis coversationroom manager  subscription error:", err)
				continue
			}

			// Durumu işle
			var managerMsg RoomManager
			err = json.Unmarshal([]byte(msg.Payload), &managerMsg)
			if err != nil {
				log.Println("coversation room  manager  unmarshal error:", err)
				continue
			}

			switch managerMsg.Type {
			case "add_player":
				sh.parentHub.BroadcastMessage(managerMsg.RoomID, (*Message)(&managerMsg.Data))
			case "kick_player":

				sh.parentHub.BroadcastMessage(managerMsg.RoomID, (*Message)(&managerMsg.Data))

			case "ban_player":
				sh.parentHub.BroadcastMessage(managerMsg.RoomID, (*Message)(&managerMsg.Data))

			case "unban_player":
				sh.parentHub.BroadcastMessage(managerMsg.RoomID, (*Message)(&managerMsg.Data))

			default:
				log.Println("Unknown conversation user manager type:", managerMsg.Type)
			}
		}
	}
}
