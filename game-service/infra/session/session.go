package session

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9" // Redis istemcisi
)

// SessionManager, oturum yönetimi işlemlerini gerçekleştiren yapı
type SessionManager struct {
	client *redis.Client // Redis istemcisi
}

// NewSessionManager, yeni bir SessionManager örneği oluşturur
// redisAddr: Redis sunucusunun adresi
func NewSessionManager(redisAddr string, password string, db int) (*SessionManager, error) {
	// Redis istemcisini oluştur
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: password, // Redis şifresi (varsayılan: boş)
		DB:       db,       // Redis veritabanı numarası (varsayılan: 0)
	})

	// Redis bağlantısını test et
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	log.Println("Connected to PostgreSQL successfully")
	return &SessionManager{
		client: client,
	}, nil
}
func (sm *SessionManager) GetRedisClient() *redis.Client {
	return sm.client
}
