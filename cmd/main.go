package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"time"

	"github.com/ignoxx/blocker/node"
	"github.com/ignoxx/blocker/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	lnAddr = ":3000"
)

func main() {
	node := node.New()

	grpcServer := grpc.NewServer()
	ln, err := net.Listen("tcp", lnAddr)
	if err != nil {
		log.Fatal(err)
	}
	proto.RegisterNodeServer(grpcServer, node)
	slog.Info("Starting gRPC server", "address", lnAddr)

	go func() {
		time.Sleep(2 * time.Second)
		makeTransaction()
	}()

	grpcServer.Serve(ln)
}

func makeTransaction() {
	client, err := grpc.NewClient(lnAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}

	c := proto.NewNodeClient(client)
	_, err = c.Handshake(
		context.TODO(),
		&proto.Version{
			Version: "test-client-v0.1",
			Height:  0,
		},
	)

	if err != nil {
		log.Fatal(err)
	}
}
