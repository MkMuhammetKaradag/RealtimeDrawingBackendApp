package config

import (
	"os"
)

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

var Services = map[string]string{
	"auth": getEnv("GATEWAY_AUTH_HTTP", "http://localhost:8081"),
	"chat": getEnv("GATEWAY_CHAT_HTTP", "http://chat-service:8082"),
	"game": getEnv("GATEWAY_GAME_HTTP", "http://localhost:8083"),
}

var ProtectedRoutes = map[string][]string{
	"auth": {
		"/logout",
		"/all-logout",
		"/hello",
	},
	"game": {
		"/create-room",
		"/join-room/:room_id",
		"/leave-room/:room_id",
	},

	"wsgame": {
		"/ws/game/:id",
	},
}

var WebSocketServices = map[string]string{
	"wsauth": getEnv("GATEWAY_AUTH_WS", "ws://auth-service:8081"),
	"wsgame": getEnv("GATEWAY_GAME_WS", "ws://localhost:8083"),
}
