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

func main() {
	makeNode(":3000")
	time.Sleep(time.Second)
	makeNode(":4000", ":3000")
	time.Sleep(5 * time.Second)
	makeNode(":5001", ":4000")

	select {}
}

func makeNode(lnAddr string, bootstrapNodes ...string) *node.Node {
	n := node.New()
	go func() {
		if err := n.Start(lnAddr, bootstrapNodes); err != nil {
			log.Fatal(err)
		}
	}()
	return n
}

func makeTransaction(addr string) {
	client, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
