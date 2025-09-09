package session

import (
	"context"
	"encoding/json"
	"log"
	"time"

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

// CreateSession, yeni bir oturum oluşturur
// ctx: Context
// userID: Kullanıcı ID'si
// token: Oturum token'ı
// userData: Kullanıcı bilgileri
// duration: Oturum süresi
func (sm *SessionManager) CreateSession(ctx context.Context, userID, token string, userData map[string]string, duration time.Duration) error {
	// Kullanıcı verilerini JSON'a dönüştür
	jsonData, err := json.Marshal(userData)
	if err != nil {
		return err
	}

	// Redis pipeline oluştur (birden fazla işlemi tek seferde yapmak için)
	pipe := sm.client.Pipeline()
	// Token'ı ve kullanıcı verilerini kaydet
	pipe.Set(ctx, token, jsonData, duration)
	// Kullanıcının oturum token'ını set'e ekle
	pipe.SAdd(ctx, "user_sessions:"+userID, token)
	// İşlemleri yürüt
	_, err = pipe.Exec(ctx)
	return err
}
