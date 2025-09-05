package config

type RateLimitConfig struct {
	Global struct {
		RequestsPerMinute int
		Burst             int
	}
	ServiceLimits map[string]ServiceLimit
}

type ServiceLimit struct {
	RequestsPerMinute int
	Burst             int
}

func NewDefaultRateLimitConfig() *RateLimitConfig {
	config := &RateLimitConfig{}

	config.Global.RequestsPerMinute = 1000
	config.Global.Burst = 100

	config.ServiceLimits = map[string]ServiceLimit{
		"auth": {
			RequestsPerMinute: 100,
			Burst:             20,
		},
	}

	return config
}
