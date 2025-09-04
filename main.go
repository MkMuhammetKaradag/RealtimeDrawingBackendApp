package main

import (
	"log"
	"net/http"

	"golang.org/x/net/websocket"
)

// clients, anlık olarak bağlı tüm WebSocket istemcilerini tutar.
// map'in değeri 'true' ile basitçe bir set olarak kullanılıyor.
var clients = make(map[*websocket.Conn]bool)

// broadcast, diğer goroutine'lardan gelen mesajları bekleyen bir kanaldır.
// Bu kanal, gelen tüm verileri diğer istemcilere dağıtmak için kullanılır.
var broadcast = make(chan interface{})

// wsHandler, her yeni WebSocket bağlantısı için çalışır.
func wsHandler(ws *websocket.Conn) {
	// Yeni istemciyi 'clients' map'ine ekle.
	clients[ws] = true

	// Fonksiyon sonlandığında (bağlantı kapandığında) istemciyi map'ten sil.
	defer func() {
		delete(clients, ws)
		ws.Close()
	}()

	var data map[string]interface{}
	// Sonsuz döngüde istemciden gelen verileri dinle.
	for {
		// JSON formatında bir veri bekle.
		if err := websocket.JSON.Receive(ws, &data); err != nil {
			// Hata olursa veya bağlantı kapanırsa döngüden çık.
			log.Println("Receive error:", err)
			return
		}
		// Gelen veriyi broadcast kanalına gönder.
		broadcast <- data
	}
}

// handleMessages, broadcast kanalını dinleyen ve mesajları tüm istemcilere yayan bir goroutine'dir.
func handleMessages() {
	for {
		// Kanaldan gelen mesajı alana kadar bekle.
		msg := <-broadcast

		// Gelen mesajın tipini kontrol et
		msgMap, ok := msg.(map[string]interface{})
		if !ok {
			log.Println("Invalid message format received")
			continue
		}

		// Eğer mesaj tipi "clear" ise, tüm istemcilere bu komutu gönder.
		if msgMap["type"] == "clear" {
			for client := range clients {
				if err := websocket.JSON.Send(client, map[string]string{"type": "clear"}); err != nil {
					log.Println("Send error:", err)
				}
			}
		} else {
			// Değilse, çizim verisi olduğunu varsay ve tüm istemcilere gönder.
			for client := range clients {
				if err := websocket.JSON.Send(client, msg); err != nil {
					log.Println("Send error:", err)
				}
			}
		}
	}
}

func main() {
	// WebSocket bağlantıları için '/ws' yolunu belirle ve wsHandler'ı ata.
	http.Handle("/ws", websocket.Handler(wsHandler))

	// 'index.html' gibi statik dosyaları sunmak için temel bir dosya sunucusu ayarla.
	// Bu, frontend tarafı için kullanılacak.
	http.Handle("/", http.FileServer(http.Dir(".")))

	// Mesajları dağıtma işini bir goroutine'e (hafif bir iş parçacığı) devret.
	// Bu sayede sunucu ana işini yapmaya devam ederken,
	// bu iş arka planda eş zamanlı olarak çalışır.
	go handleMessages()

	log.Println("Sunucu 8080 portunda başlatıldı...")
	// Sunucuyu 8080 portunda başlat ve gelen istekleri dinle.
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
