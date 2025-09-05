package utils

import (
	"fmt"
	"gateway-service/internal/config"
	"log"
	"net/http"
	"strings"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"github.com/google/uuid"
	gorilla "github.com/gorilla/websocket"
)

func isProtectedRoute(service, path string) bool {
	// Servisin korumalı route listesini al
	protectedList, ok := config.ProtectedRoutes[service]
	if !ok {
		return false
	}
	// Path'in korumalı route'lardan biriyle başlayıp başlamadığını kontrol et
	for _, p := range protectedList {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
} // BuildProxyHandler, verilen servis için bir Fiber handler döndürür.
func BuildProxyHandler(serviceName string) fiber.Handler {
	var serviceURL string

	serviceURL = config.Services[serviceName]

	if serviceURL == "" {
		return func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Service not found"})
		}
	}

	return func(c *fiber.Ctx) error {
		// Orijinal URL'den servis prefix'ini kaldır
		rewrittenPath := strings.TrimPrefix(c.Path(), "/"+serviceName)
		if rewrittenPath == "" {
			rewrittenPath = "/"
		}

		// Sorgu parametrelerini al
		queryString := c.Context().URI().QueryString()

		// Hedef URL'i oluştur ve sorgu parametrelerini ekle
		fullURL := serviceURL + rewrittenPath
		if len(queryString) > 0 {
			fullURL += "?" + string(queryString)
		}

		// Debug için logla
		log.Printf("Proxying request from %s to %s", c.OriginalURL(), fullURL)

		// Fiber'ın kendi proxy'sini kullanarak isteği yönlendir
		return proxy.Do(c, fullURL)
	}
}

type WrbsocketHandler = func(*websocket.Conn)

func BuildWebSocketProxy(serviceName string) WrbsocketHandler {

	var serviceURL string

	serviceURL = config.WebSocketServices[serviceName]
	if serviceURL == "" {
		return func(c *websocket.Conn) {

		}

	}
	return func(clientConn *websocket.Conn) {
		// Hedef path'i al
		target := clientConn.Locals("ws_path")
		targetPath := ""
		if target != nil {
			targetPath = "/" + target.(string)
		}
		url := config.WebSocketServices[serviceName] + targetPath
		fmt.Println("url>", url)
		// İstek başlıklarını hazırla
		requestHeaders := http.Header{}
		headerKeys := []string{"Authorization", "Session"}

		for _, key := range headerKeys {
			value := clientConn.Headers(key)
			if value != "" {
				requestHeaders.Set(key, value)
			}
		}

		// Session cookie'sini ekle
		cookie := clientConn.Cookies("Session")
		if cookie != "" {
			requestHeaders.Set("Cookie", "Session="+cookie)
		}

		// Backend'e WebSocket bağlantısı kur
		dialer := &gorilla.Dialer{}
		backendConn, _, err := dialer.Dial(url, requestHeaders)
		if err != nil {
			log.Printf("WS backend error: %v", err)
			return
		}

		defer backendConn.Close()

		// Benzersiz client ID oluştur
		clientID := uuid.New().String()

		// İletişimi durdurmak için kanal
		stop := make(chan struct{})

		// Client -> Backend iletişimi
		go func() {
			defer func() {
				log.Printf("client -> backend closed [clientID: %s]", clientID)
				_ = backendConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "client closed"))
				close(stop)
			}()
			for {
				// Client'tan mesaj oku
				t, msg, err := clientConn.ReadMessage()
				if err != nil {
					if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
						log.Printf("client normal close [clientID: %s]", clientID)
					} else {
						log.Printf("client read error [clientID: %s]: %v", clientID, err)
					}
					return
				}
				// Backend'e mesaj gönder
				err = backendConn.WriteMessage(t, msg)
				if err != nil {
					log.Printf("backend write error [clientID: %s]: %v", clientID, err)
					return
				}
			}
		}()

		// Backend -> Client iletişimi
		go func() {
			defer func() {
				log.Printf("backend -> client closed [clientID: %s]", clientID)
				_ = clientConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "backend closed"))
			}()

			for {
				// Backend'den mesaj oku
				t, msg, err := backendConn.ReadMessage()
				if err != nil {
					log.Printf("backend read error [clientID: %s]: %v", clientID, err)
					return
				}
				// Client'a mesaj gönder
				err = clientConn.WriteMessage(t, msg)
				if err != nil {
					log.Printf("client write error [clientID: %s]: %v", clientID, err)
					return
				}
			}
		}()

		// İletişim durana kadar bekle
		<-stop
	}
}
