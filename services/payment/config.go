package main

import "os"

// config is read directly from env — this is a small standalone service,
// not worth pulling in viper for six values.
type config struct {
	GRPCAddr            string
	HTTPAddr            string
	XenditSecretKey     string
	XenditCallbackToken string
	Mock                bool
	MonolithCallbackURL string
	InternalSecret      string
}

func loadConfig() config {
	return config{
		GRPCAddr:            getenv("GRPC_ADDR", ":50051"),
		HTTPAddr:            getenv("HTTP_ADDR", ":8081"),
		XenditSecretKey:     os.Getenv("XENDIT_SECRET_KEY"),
		XenditCallbackToken: os.Getenv("XENDIT_CALLBACK_TOKEN"),
		Mock:                getenv("MOCK", "true") == "true",
		MonolithCallbackURL: getenv("MONOLITH_CALLBACK_URL", "http://localhost:8080/internal/payments/callback"),
		InternalSecret:      os.Getenv("INTERNAL_SECRET"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
