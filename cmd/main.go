package main

import (
	"log"
	"time"

	"github.com/ignoxx/blocker/crypto"
	"github.com/ignoxx/blocker/node"
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
	time.Sleep(500 * time.Millisecond)
	makeNode(":4000", &privKey2, chain2, validators, ":3000")
	time.Sleep(500 * time.Millisecond)
	makeNode(":5001", &privKey3, chain3, validators, ":3000")

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
