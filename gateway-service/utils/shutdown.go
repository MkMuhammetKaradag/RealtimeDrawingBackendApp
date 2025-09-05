// Package utils, API Gateway'in yardımcı fonksiyonlarını içerir
package utils

import (
	"log"       // Loglama için
	"os"        // İşletim sistemi işlemleri için
	"os/signal" // Sinyal işlemleri için
	"syscall"   // Sistem çağrıları için

	"github.com/gofiber/fiber/v2" // Web framework
)

// GracefulShutdown, uygulamanın düzgün bir şekilde kapatılmasını sağlar
// app: Fiber uygulaması
func GracefulShutdown(app *fiber.App) {
	// Sinyal kanalı oluştur
	sig := make(chan os.Signal, 1)
	// SIGINT (Ctrl+C) ve SIGTERM sinyallerini dinle
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	// Sinyal gelene kadar bekle
	<-sig
	// Kapatma işlemini başlat
	log.Println("Shutting down...")
	// Uygulamayı düzgün bir şekilde kapat
	app.Shutdown()
}
