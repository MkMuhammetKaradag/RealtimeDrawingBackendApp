package bootstrap

import (
	"auth-service/config"
	authhandler "auth-service/internal/api/handler"
	"auth-service/internal/handler"
	"auth-service/internal/server"
	"fmt"
	"log"

	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

type Config struct {
	Port         string
	IdleTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func SetupServer(config config.Config, httpHandlers map[string]interface{}) *fiber.App {

	serverConfig := server.Config{
		Port:         config.Server.Port,
		IdleTimeout:  5 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	app := server.NewFiberApp(serverConfig)
	signUpHandler := httpHandlers["signup"].(*authhandler.SignUpHandler)
	activateHandler := httpHandlers["activate"].(*authhandler.ActivateHandler)
	signInHandler := httpHandlers["signin"].(*authhandler.SignInHandler)
	logoutHandler := httpHandlers["logout"].(*authhandler.LogoutHandler)
	allLogoutHandler := httpHandlers["all-logout"].(*authhandler.AllLogoutHandler)

	app.Post("/signup", handler.HandleBasic[authhandler.SignUpRequest, authhandler.SignUpResponse](signUpHandler))
	app.Post("/user-activate", handler.HandleBasic[authhandler.ActivateRequest, authhandler.ActivateResponse](activateHandler))
	app.Post("/signin", handler.HandleWithFiber[authhandler.SignInRequest, authhandler.SignInResponse](signInHandler))
	app.Post("/logout", handler.HandleWithFiber[authhandler.LogoutRequest, authhandler.LogoutResponse](logoutHandler))
	app.Post("/all-logout", handler.HandleWithFiber[authhandler.AllLogoutRequest, authhandler.AllLogoutResponse](allLogoutHandler))
	app.Get("/hello", func(c *fiber.Ctx) error {

		// İstekteki 'my_cookie' isimli cookie'nin değerini al.
		// Eğer cookie yoksa, boş bir string döner.
		myCookie := c.Cookies("my_cookie")

		// İstekteki tüm cookie'leri döngüyle okuyup konsola yazdır.
		// Bu, gönderilen tüm cookie'leri görmek için kullanışlıdır.
		log.Println("Gelen tüm cookie'ler:")
		c.Request().Header.VisitAllCookie(func(key, value []byte) {
			log.Printf("Cookie Adı: %s, Değer: %s\n", key, value)
		})

		// Eğer belirli bir cookie varsa, değerini JSON yanıtına ekle.
		if myCookie != "" {
			return c.JSON(fiber.Map{
				"message":   "Hello from Auth Service!",
				"my_cookie": myCookie,
				"info":      "Gönderdiğiniz cookie başarıyla alındı!",
			})
		}

		// Eğer belirli bir cookie yoksa, farklı bir JSON yanıtı döndür.
		return c.JSON(fiber.Map{
			"message": "Hello from Auth Service!",
			"info":    "Belirtilen cookie bulunamadı.",
		})
	})

	app.Get("/data", func(c *fiber.Ctx) error {
		// --- Params (Sorgu Parametreleri) ---
		// Tüm sorgu parametrelerini bir Map'e al.
		queryParams := c.Queries()

		// --- Headers (Başlıklar) ---
		// Bir başlığa erişim.
		myHeader := c.Get("my_header")

		// Tüm başlıkları döngüyle al.
		headersMap := make(map[string]string)
		c.Request().Header.VisitAll(func(key, value []byte) {
			headersMap[string(key)] = string(value)
		})

		// --- Cookies (Çerezler) ---
		// Belirli bir çereze erişim.
		myCookie := c.Cookies("my_cookie")

		// Yanıt olarak tüm verileri JSON formatında gönder.
		return c.JSON(fiber.Map{
			"message": "İsteğinizdeki tüm veriler alındı!",
			"request_data": fiber.Map{
				"query_params":    queryParams,
				"headers":         headersMap,
				"my_header_value": myHeader,
				"my_cookie_value": myCookie,
			},
		})
	})

	//WebSocket bağlantıları için middleware
	app.Use("/ws", func(c *fiber.Ctx) error {
		// İstemci WebSocket protokolüne yükseltme istedi mi kontrol et
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// WebSocket endpoint'i
	app.Get("/ws/:id", websocket.New(func(c *websocket.Conn) {
		// Bağlantı bilgilerini yazdır
		fmt.Println(c.Locals("allowed"))
		fmt.Println(c.Params("id"))
		fmt.Println(c.Query("v"))
		fmt.Println("headders:", c.Headers("Session"))
		fmt.Println("cookie", c.Cookies("Session"))

		var (
			mt  int
			msg []byte
			err error
		)

		// Bağlantı kapatıldığında sinyal göndermek için kanal
		done := make(chan struct{})

		// Bağlantı kapatma işleyicisi
		c.SetCloseHandler(func(code int, text string) error {
			fmt.Printf("websocket closed: code=%d, reason=%s\n", code, text)
			close(done) // ticker'ı durdurmak için
			return nil
		})

		// Ping-pong mekanizması: Her 5 saniyede bir "hello" mesajı gönder
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					if err := c.WriteMessage(websocket.TextMessage, []byte("hello")); err != nil {
						fmt.Println("write hello error:", err)
						return
					}
				}
			}
		}()

		// Ana mesaj döngüsü
		for {
			// Mesaj oku
			if mt, msg, err = c.ReadMessage(); err != nil {
				fmt.Println("read error:", err)
				break
			}
			fmt.Printf("recv: %s\n", msg)

			// Gelen mesajı geri gönder (echo)
			if err = c.WriteMessage(mt, msg); err != nil {
				fmt.Println("write echo error:", err)
				break
			}
		}

		// Ekstra güvenlik için done kanalını tekrar kapatmayı dene
		select {
		case <-done:
		default:
			close(done)
		}
	}))
	return app
}
