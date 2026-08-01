// Command notification is the standalone notification-service (FR-6.6): a
// gRPC server wrapping the Mailjet Send API. Unlike payment-service, it has
// no inbound webhook — Mailjet doesn't need a callback relay.
package main

import (
	"net"
	"os"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"

	notificationv1 "erdinhrmwn/bangunin/proto/notification/v1"
)

func main() {
	log := zerolog.New(os.Stdout).With().Timestamp().Str("service", "notification").Logger()
	cfg := loadConfig()

	mailjet := newMailjetClient(cfg.MailjetAPIKey, cfg.MailjetSecret, cfg.MailjetFromAddr, cfg.MailjetFromName, cfg.Mock)

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatal().Err(err).Msg("grpc listen failed")
	}

	grpcServer := grpc.NewServer()
	notificationv1.RegisterNotificationServiceServer(grpcServer, &notificationServer{mailjet: mailjet, log: log})

	log.Info().Str("addr", cfg.GRPCAddr).Bool("mock", cfg.Mock).Msg("grpc server starting")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal().Err(err).Msg("grpc server failed")
	}
}
