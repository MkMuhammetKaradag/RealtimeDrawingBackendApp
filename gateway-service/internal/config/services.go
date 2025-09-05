package config

var Services = map[string]string{
	"auth":         "http://localhost:8081",
	"chat":         "http://localhost:8082",
}
var ProtectedRoutes = map[string][]string{
	"auth": {
		"/logout",
	},
}

var WebSocketServices = map[string]string{
	"auth": "ws://localhost:8081/ws",
}
