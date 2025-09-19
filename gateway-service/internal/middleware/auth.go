package middleware

import (
	"encoding/json"
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

		if isProtected(serviceName, strings.TrimPrefix(path, "/"+serviceName)) {
			token := c.Cookies("Session")
			if token == "" {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
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

func isProtected(serviceName, path string) bool {
	protectedList, ok := config.ProtectedRoutes[serviceName]
	if !ok {
		return false
	}
	for _, p := range protectedList {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
