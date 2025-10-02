package middleware

import (
	"encoding/json"
	"fmt"
	"gateway-service/internal/config"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type AuthServiceResponse struct {
	UserID string `json:"user_id"`
}

func AuthGuard() fiber.Handler {
	return func(c *fiber.Ctx) error {
		serviceName, _ := c.Locals("service_name").(string)
		path := c.Path()
		var refreshNeededHeader string
		isProt := isProtected(c, serviceName, path)
		if isProt {
			var token string

			if strings.Contains(c.Get("Connection"), "Upgrade") && c.Get("Upgrade") == "websocket" {
				// WebSocket bağlantıları için token'ı query parametrelerinden veya Authorization header'ından al
				token = c.Query("token")
				fmt.Println("WebSocket bağlantısı tespit edildi, token kontrol ediliyor...", token)
				fmt.Println("cookie", c.Cookies("Session"))
				if token == "" {
					token = c.Get("Session")
					if token == "" {
						token = c.Cookies("Session")
					}
				}
				if token == "" {
					fmt.Println("WebSocket token veya Authorization header bulunamadı")
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized: missing token or Authorization for WebSocket"})
				}
			} else {
				token = c.Cookies("Session")
				if token == "" {
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
				}
			}

			req, err := http.NewRequest("GET", "http://localhost:8081/validate-token", nil)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
			}

			req.AddCookie(&http.Cookie{Name: "Session", Value: token})

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "auth service is unavailable"})
			}

			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := ioutil.ReadAll(resp.Body)
				return c.Status(resp.StatusCode).Send(body)
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to read auth service response"})
			}

			var authResp AuthServiceResponse
			if err := json.Unmarshal(body, &authResp); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to parse auth service response"})
			}

			// 2. Kullanıcı ID'sini al
			userID := authResp.UserID

			if userID == "" {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user ID not found in auth response"})
			}

			// 3. Kullanıcı ID'sini sonraki handler'lar için Fiber'in yerel değişkenlerine ekle
			c.Locals("user_id", userID)
			// Auth servisinden gelen "x-refresh-needed" başlığını sakla
			if refreshNeeded := resp.Header.Get("X-Refresh-Needed"); refreshNeeded == "true" {
				refreshNeededHeader = "true"
			}
		}

		// İsteği sonraki handler'a devret
		// Bu, orijinal isteği hedef servise yönlendirir.
		if err := c.Next(); err != nil {
			return err
		}

		// Eğer refreshNeededHeader değeri "true" ise, nihai yanıta ekle
		if refreshNeededHeader == "true" {
			c.Response().Header.Set("X-Refresh-Needed", "true")
			c.Response().Header.Set("Access-Control-Expose-Headers", "X-Refresh-Needed")
		}

		return nil
	}
}

func isProtected(c *fiber.Ctx, serviceName, fullPath string) bool {
	// serviceName'e göre korumalı rotaları al
	protectedList, ok := config.ProtectedRoutes[serviceName]
	if !ok {
		return false // Servis korumalı rota listesinde yoksa, korumalı değil
	}

	// Gelen yolun sadece servis adından sonraki kısmını al
	path := strings.TrimPrefix(fullPath, "/"+serviceName)

	// Korumalı rota listesindeki her bir desenle eşleşmeye çalış
	for _, pattern := range protectedList {
		// Fiber'in Path.Match'ini kullanmak en iyi yoldur
		// Bu, ":id" gibi parametreleri doğru şekilde ele alır
		if strings.Contains(pattern, ":") {
			// Eğer desen parametre içeriyorsa, Fiber'in Match metodunu kullan
			// Match metodunun kendi içinde dinamik segmentleri kontrol etme yeteneği vardır.
			// Ancak, Path.Match kullanabilmek için rota tanımına ihtiyaç duyarız.
			// Basitlik ve Fiber'dan bağımsızlık adına, string yerine daha genel bir yaklaşım deneyelim.
			// Basit bir geçici çözüm:
			// '/game/:id' -> '/game/'
			// '/game/12347' -> '/game/'

			// Daha gelişmiş ve doğru çözüm: Fiber'in kendi yönlendirme motorunu kullanmak
			// Burada basit bir mantık ile kontrol sağlıyoruz.
			// Örnek: "/game/:id" desenini "/game/12347" yoluyla karşılaştır.
			// Bu, '/game/' ön ekinin uyuştuğunu kontrol etmek için yeterli olabilir.
			if strings.HasPrefix(path, strings.Split(pattern, ":")[0]) {
				return true
			}

		} else if strings.HasPrefix(path, pattern) {
			// Parametresiz rotalar için eski mantık çalışmaya devam edebilir
			return true
		}
	}
	return false
}
