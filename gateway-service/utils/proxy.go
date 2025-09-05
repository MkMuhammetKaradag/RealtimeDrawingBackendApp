package utils

import (
	"bytes"
	"gateway-service/internal/config"

	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
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
}

func BuildProxyHandler(service string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals("service_name", service) // Servis adını context'e ekle

		// Orijinal URL'den servis prefix'ini kaldır
		path := strings.TrimPrefix(c.OriginalURL(), "/"+service)

		// HTTP istemcisi oluştur (10 saniye timeout ile)
		client := &http.Client{Timeout: 10 * time.Second}
		targetURL := config.Services[service]
		fullURL := targetURL + path

		// Query parametrelerini ekle
		if query := c.Context().URI().QueryString(); len(query) > 0 {
			fullURL += "?" + string(query)
		}

		// Yeni HTTP isteği oluştur
		req, err := http.NewRequest(c.Method(), fullURL, bytes.NewReader(c.Body()))
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		// İstek başlıklarını kopyala
		for k, vals := range c.GetReqHeaders() {
			for _, v := range vals {
				req.Header.Add(k, v)
			}
		}

		// İsteği gönder
		resp, err := client.Do(req)
		if err != nil {
			return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
		}
		defer resp.Body.Close()

		// Yanıt gövdesini oku
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		// Yanıt başlıklarını kopyala
		for k, vals := range resp.Header {
			for _, v := range vals {

				if strings.ToLower(k) == "set-cookie" {
					c.Append("Set-Cookie", v) // Append, çünkü Fiber tek başlıkta birden fazla Set-Cookie'yi desteklemez
				} else {
					c.Set(k, v)
				}
			}
		}

		// Yanıtı gönder
		return c.Status(resp.StatusCode).Send(body)
	}
}
