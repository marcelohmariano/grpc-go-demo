package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/marcelohmariano/grpc-go-demo/internal/note"
)

var (
	grpcAddr = flag.String("grpc", "localhost:50051", "grpc listen address")
	httpAddr = flag.String("http", "localhost:8080", "http listen address")
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background())
	defer stop()

	s := note.NewAPIServer(*grpcAddr, *httpAddr)
	return s.Listen(ctx)
}
