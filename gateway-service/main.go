package main

import (
	"gateway-service/internal/gateway"
	"log"
)

func main() {
	server, err := gateway.NewGatewayServer()
	if err != nil {
		log.Fatalf("Failed to create gateway server: %v", err)
	}

	port := "8080"

	log.Fatal(server.Start(port))

}
