package main

import "os"

// config is read directly from env — this is a small standalone service,
// not worth pulling in viper for a handful of values.
type config struct {
	GRPCAddr        string
	MailjetAPIKey   string
	MailjetSecret   string
	MailjetFromAddr string
	MailjetFromName string
	Mock            bool
}

func loadConfig() config {
	return config{
		GRPCAddr:        getenv("GRPC_ADDR", ":50052"),
		MailjetAPIKey:   os.Getenv("MAILJET_API_KEY"),
		MailjetSecret:   os.Getenv("MAILJET_SECRET"),
		MailjetFromAddr: getenv("MAILJET_FROM_ADDR", "no-reply@bangunin.local"),
		MailjetFromName: getenv("MAILJET_FROM_NAME", "Bangunin"),
		Mock:            getenv("MOCK", "true") == "true",
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
