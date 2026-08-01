// Command payment is the standalone payment-service (FR-5.5): a gRPC server
// wrapping the Xendit Invoice API, plus an HTTP endpoint receiving Xendit's
// webhook and relaying it to the monolith's internal callback endpoint.
package main

import (
	"net"
	"net/http"
	"os"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"

	paymentv1 "erdinhrmwn/bangunin/proto/payment/v1"
)

func main() {
	log := zerolog.New(os.Stdout).With().Timestamp().Str("service", "payment").Logger()
	cfg := loadConfig()

	xendit := newXenditClient(cfg.XenditSecretKey, cfg.Mock)

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/callback", &callbackHandler{cfg: cfg, log: log})
		log.Info().Str("addr", cfg.HTTPAddr).Msg("http callback listener starting")
		if err := http.ListenAndServe(cfg.HTTPAddr, mux); err != nil {
			log.Fatal().Err(err).Msg("http server failed")
		}
	}()

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatal().Err(err).Msg("grpc listen failed")
	}

	grpcServer := grpc.NewServer()
	paymentv1.RegisterPaymentServiceServer(grpcServer, &paymentServer{xendit: xendit, log: log})

	log.Info().Str("addr", cfg.GRPCAddr).Bool("mock", cfg.Mock).Msg("grpc server starting")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal().Err(err).Msg("grpc server failed")
	}
}
