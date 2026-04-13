package main

import (
	"context"
	"log"
	"time"

	"github.com/ignoxx/blocker/crypto"
	"github.com/ignoxx/blocker/node"
	"github.com/ignoxx/blocker/proto"
	"github.com/ignoxx/blocker/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	makeNode(":3000")
	time.Sleep(time.Second)
	makeNode(":4000", ":3000")
	time.Sleep(5 * time.Second)
	makeNode(":5001", ":4000")

	time.Sleep(2 * time.Second)
	makeTransaction(":3000")

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
	privKey, _ := crypto.GeneratePrivateKey()
	pubKey := privKey.Public()

	tx := &proto.Transaction{
		Version: 1,
		Inputs: []*proto.TxInput{
			{
				PrevTxHash:   util.RandomHash(),
				PrevOutIndex: 0,
				PublicKey:    pubKey.Bytes(),
			},
		},
		Outputs: []*proto.TxOutput{
			{
				Amount:  99,
				Address: pubKey.Address().Bytes(),
			},
		},
	}

	_, err = c.HandleTransaction(context.TODO(), tx)

	if err != nil {
		log.Fatal(err)
	}
}
