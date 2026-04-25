package main

import (
	"context"
	"encoding/hex"
	"log"
	"time"

	"github.com/ignoxx/blocker/crypto"
	"github.com/ignoxx/blocker/node"
	"github.com/ignoxx/blocker/proto"
	"github.com/ignoxx/blocker/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	privKey1, _ := crypto.GeneratePrivateKey()
	privKey2, _ := crypto.GeneratePrivateKey()
	privKey3, _ := crypto.GeneratePrivateKey()

	validators := node.NewValidatorSet([]*crypto.PublicKey{
		privKey1.Public(),
		privKey2.Public(),
		privKey3.Public(),
	})

	// Each node gets its OWN chain with its OWN stores — standalone!
	chain1 := node.NewChain(node.NewMemoryBlockStore(), node.NewMemoryTxStore(), validators)
	chain2 := node.NewChain(node.NewMemoryBlockStore(), node.NewMemoryTxStore(), validators)
	chain3 := node.NewChain(node.NewMemoryBlockStore(), node.NewMemoryTxStore(), validators)

	makeNode(":3000", &privKey1, chain1, validators)
	time.Sleep(time.Second)
	makeNode(":4000", &privKey2, chain2, validators, ":3000")
	time.Sleep(2 * time.Second)
	makeNode(":5001", &privKey3, chain3, validators, ":4000")

	// Wait for chain to produce a few blocks, then submit a tx
	time.Sleep(12 * time.Second)
	makeTransaction(":3000")

	select {}
}

func makeNode(lnAddr string, privKey *crypto.PrivateKey, chain *node.Chain, validators *node.ValidatorSet, bootstrapNodes ...string) *node.Node {
	cfg := node.ServerConfig{
		Version:    "blocker-v1",
		ListenAddr: lnAddr,
		PrivateKey: privKey,
	}

	n := node.NewNode(cfg, chain, validators)
	go func() {
		if err := n.Start(bootstrapNodes); err != nil {
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

	genesisPrivKey, _ := crypto.NewPrivateKeyFromSeedStr("0f71b4d49b833860f40ef07c39556287a239c1a27bc26fd5261b99f56af867f2")
	genesisPubKey := genesisPrivKey.Public()

	h, _ := hex.DecodeString("a57f3604f7ef10fb992523eb9b9ecf2022a462b1caea9b5bc74207a66fd28b95")
	tx := &proto.Transaction{
		Version: 1,
		Inputs: []*proto.TxInput{
			{
				PrevTxHash:   h,
				PrevOutIndex: 0,
				PublicKey:    genesisPubKey.Bytes(),
			},
		},
		Outputs: []*proto.TxOutput{
			{
				Amount:  1,
				Address: genesisPubKey.Address().Bytes(),
			},
			{
				Amount:  999,
				Address: genesisPubKey.Address().Bytes(),
			},
		},
	}

	sig := types.SignTransaction(&genesisPrivKey, tx)
	tx.Inputs[0].Signature = sig.Bytes()

	_, err = c.HandleTransaction(context.TODO(), tx)
	if err != nil {
		log.Println("tx error:", err)
	}
}
