package grpcclient

import (
	"crypto/tls"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// transportCredentials returns plaintext credentials only in development
// (local docker-compose, no TLS between containers); every other env
// requires TLS so gRPC traffic isn't sent in cleartext.
func transportCredentials(env string) credentials.TransportCredentials {
	if env == "development" {
		return insecure.NewCredentials()
	}
	return credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})
}
