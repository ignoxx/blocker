package main

import (
	"context"
	"log"
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
	makeNode(lnAddr)
	makeNode(":4000", lnAddr)

	go func() {
		time.Sleep(2 * time.Second)
		makeTransaction()
	}()

	select{}
}

func makeNode(lnAddr string, bootstrapNodes ...string) *node.Node {
	n := node.New()
	go n.Start(lnAddr)
	if len(bootstrapNodes) > 0 {
		if err := n.BootstrapNetwork(bootstrapNodes); err != nil {
			log.Fatal(err)
		}
	}
	return n
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
			Version:    "test-client-v0.1",
			Height:     0,
			ListenAddr: "test-client:3001",
		},
	)

	if err != nil {
		log.Fatal(err)
	}
}
