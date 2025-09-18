package session

import (
	"auth-service/domain"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// SessionData, oturumda saklanacak kullanıcı verilerini temsil eder.
// type SessionData struct {
// 	UserID   string `json:"userID"`
// 	Username string `json:"username"`
// 	Device   string `json:"device"`
// 	Ip       string `json:"ip"`
// 	// Diğer kullanıcı verilerini buraya ekleyebilirsiniz.
// }

// SessionManager, Redis tabanlı oturum yönetimi sağlar.
type SessionManager struct {
	client *redis.Client
}

// NewSessionManager, yeni bir SessionManager örneği oluşturur ve Redis'e bağlantıyı test eder.
func NewSessionManager(redisAddr string, password string, db int) (*SessionManager, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: password,
		DB:       db,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Println("Successfully connected to Redis.")

	return &SessionManager{
		client: client,
	}, nil
}

// CreateSession, yeni bir oturum oluşturur.
func (sm *SessionManager) CreateSession(ctx context.Context, token string, data *domain.SessionData, duration time.Duration) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	pipe := sm.client.Pipeline()
	pipe.Set(ctx, token, jsonData, duration)
	pipe.SAdd(ctx, sm.userSessionsKey(data.UserID), token)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// GetSession, belirtilen token'a ait oturum verilerini alır.
func (sm *SessionManager) GetSession(ctx context.Context, token string) (*domain.SessionData, error) {
	val, err := sm.client.Get(ctx, token).Result()
	if err == redis.Nil {
		return nil, nil // Oturum bulunamadı
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session data for token %s: %w", token, err)
	}

	var data domain.SessionData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session data for token %s: %w", token, err)
	}

	return &data, nil
}

// DeleteSession, belirtilen token'a ait oturumu siler.
func (sm *SessionManager) DeleteSession(ctx context.Context, token string) error {
	// İlk olarak, session verilerini alarak kullanıcı ID'sini bulmalıyız.
	sessionData, err := sm.GetSession(ctx, token)
	if err != nil {
		return err
	}
	if sessionData == nil {
		return nil // Zaten silinmiş
	}

	pipe := sm.client.Pipeline()
	pipe.Del(ctx, token)
	pipe.SRem(ctx, sm.userSessionsKey(sessionData.UserID), token)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// DeleteAllUserSessions, bir kullanıcıya ait tüm oturumları siler.
func (sm *SessionManager) DeleteAllUserSessions(ctx context.Context, token string) error {
	sessionData, err := sm.GetSession(ctx, token)
	if err != nil {
		return err
	}
	if sessionData == nil {
		return nil // Zaten silinmiş
	}

	sessionSetKey := sm.userSessionsKey(sessionData.UserID)
	tokens, err := sm.client.SMembers(ctx, sessionSetKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get session tokens for user %s: %w", sessionData.UserID, err)
	}

	if len(tokens) == 0 {
		return nil
	}

	pipe := sm.client.Pipeline()
	// Del komutu birden fazla anahtarı aynı anda silme yeteneğine sahiptir.
	keysToDelete := make([]string, 0, len(tokens)+1)
	keysToDelete = append(keysToDelete, tokens...)
	keysToDelete = append(keysToDelete, sessionSetKey)

	pipe.Del(ctx, keysToDelete...)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete all sessions for user %s: %w", sessionData.UserID, err)
	}

	return nil
}

// userSessionsKey, kullanıcı oturum kümesi için Redis anahtarını oluşturur.
func (sm *SessionManager) userSessionsKey(userID string) string {
	return fmt.Sprintf("auth-service:user_sessions:%s", userID)
}

// UpdateSession, eski token'ı siler ve yeni token ile yeni bir oturum oluşturur.
func (sm *SessionManager) UpdateSession(ctx context.Context, oldToken, newToken string, data *domain.SessionData, duration time.Duration) error {
	pipe := sm.client.Pipeline()
	// Eski token'ı sil
	pipe.Del(ctx, oldToken)
	// Yeni token ile yeni bir oturum oluştur
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}
	pipe.Set(ctx, newToken, jsonData, duration)
	// Kullanıcının oturum setini güncelle
	pipe.SRem(ctx, sm.userSessionsKey(data.UserID), oldToken)
	pipe.SAdd(ctx, sm.userSessionsKey(data.UserID), newToken)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}
