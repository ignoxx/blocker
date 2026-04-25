package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/ignoxx/blocker/crypto"
	"github.com/ignoxx/blocker/proto"
	"github.com/ignoxx/blocker/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var nodeAddr = flag.String("node", ":3000", "node grpc address")

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		usage()
		return
	}

	switch args[0] {
	case "create":
		createWallet()
	case "balance":
		if len(args) < 2 {
			fmt.Println("Usage: cli balance <address>")
			os.Exit(1)
		}
		getBalance(*nodeAddr, args[1])
	case "txs":
		if len(args) < 2 {
			fmt.Println("Usage: cli txs <address>")
			os.Exit(1)
		}
		getTransactions(*nodeAddr, args[1])
	case "send":
		if len(args) < 4 {
			fmt.Println("Usage: cli send <from_priv_key> <to_address> <amount>")
			os.Exit(1)
		}
		sendCoins(*nodeAddr, args[1], args[2], args[3])
	default:
		usage()
	}
}

func usage() {
	fmt.Println("Usage: cli [options] <command> [args]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  create                                    Create a new wallet")
	fmt.Println("  balance <address>                         Query balance for an address")
	fmt.Println("  txs <address>                             Query transactions for an address")
	fmt.Println("  send <from_priv_key> <to_address> <amt>   Send coins")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -node string                              Node gRPC address (default \":3000\")")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  cli create")
	fmt.Println("  cli balance a0a88f960d97ac2c8b3753270ef6a5df9f3efd1c")
	fmt.Println("  cli send 0f71b4d... a0a88f9... 100")
}

func createWallet() {
	privKey, err := crypto.GeneratePrivateKey()
	if err != nil {
		log.Fatal(err)
	}

	addr := privKey.Public().Address()

	seed := privKey.String()[:64]

	fmt.Println("=== New Wallet ===")
	fmt.Println("Private Key (seed):", seed)
	fmt.Println("Public Key:        ", privKey.Public().String())
	fmt.Println("Address:           ", addr.String())
	fmt.Println()
	fmt.Println("Save your private key — it cannot be recovered!")
}

func getBalance(nodeAddr, addrStr string) {
	addr, err := hex.DecodeString(addrStr)
	if err != nil {
		log.Fatal("invalid address:", err)
	}

	client, err := grpc.NewClient(nodeAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	c := proto.NewNodeClient(client)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.GetBalance(ctx, &proto.GetBalanceRequest{Address: addr})
	if err != nil {
		log.Fatal("query failed:", err)
	}

	fmt.Println("Balance:", resp.Balance)
}

func getTransactions(nodeAddr, addrStr string) {
	addr, err := hex.DecodeString(addrStr)
	if err != nil {
		log.Fatal("invalid address:", err)
	}

	client, err := grpc.NewClient(nodeAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	c := proto.NewNodeClient(client)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.GetTransactions(ctx, &proto.GetTransactionsRequest{Address: addr})
	if err != nil {
		log.Fatal("query failed:", err)
	}

	if len(resp.Transactions) == 0 {
		fmt.Println("No transactions found.")
		return
	}

	fmt.Printf("Found %d transaction(s):\n\n", len(resp.Transactions))
	for i, tx := range resp.Transactions {
		fmt.Printf("--- Tx %d ---\n", i+1)
		fmt.Printf("  Version: %d\n", tx.Version)
		fmt.Printf("  Inputs:  %d\n", len(tx.Inputs))
		for j, in := range tx.Inputs {
			fmt.Printf("    [%d] PrevTx: %s, Index: %d\n", j, hex.EncodeToString(in.PrevTxHash), in.PrevOutIndex)
		}
		fmt.Printf("  Outputs: %d\n", len(tx.Outputs))
		for j, out := range tx.Outputs {
			fmt.Printf("    [%d] Amount: %d -> %s\n", j, out.Amount, hex.EncodeToString(out.Address))
		}
		fmt.Println()
	}
}

func sendCoins(nodeAddr, fromPrivKeyHex, toAddrHex, amountStr string) {
	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		log.Fatal("invalid amount:", err)
	}

	privKey, err := crypto.NewPrivateKeyFromSeedStr(fromPrivKeyHex)
	if err != nil {
		log.Fatal("invalid private key:", err)
	}

	toAddr, err := hex.DecodeString(toAddrHex)
	if err != nil {
		log.Fatal("invalid recipient address:", err)
	}

	client, err := grpc.NewClient(nodeAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	c := proto.NewNodeClient(client)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Fetch UTXOs for the sender
	resp, err := c.GetUTXOs(ctx, &proto.GetUTXOsRequest{Address: privKey.Public().Address().Bytes()})
	if err != nil {
		log.Fatal("failed to fetch UTXOs:", err)
	}

	if len(resp.Utxos) == 0 {
		log.Fatal("no UTXOs available to spend")
	}

	// Simple strategy: spend the first UTXO (for learning project)
	// Real wallets would select UTXOs optimally
	utxo := resp.Utxos[0]

	if utxo.Amount < amount {
		log.Fatalf("insufficient funds: have %d, need %d", utxo.Amount, amount)
	}

	prevHash, err := hex.DecodeString(utxo.Hash)
	if err != nil {
		log.Fatal("invalid utxo hash:", err)
	}

	tx := &proto.Transaction{
		Version: 1,
		Inputs: []*proto.TxInput{
			{
				PrevTxHash:   prevHash,
				PrevOutIndex: uint32(utxo.Index),
				PublicKey:    privKey.Public().Bytes(),
			},
		},
		Outputs: []*proto.TxOutput{
			{
				Amount:  amount,
				Address: toAddr,
			},
		},
	}

	// Add change output if there's leftover
	if utxo.Amount > amount {
		tx.Outputs = append(tx.Outputs, &proto.TxOutput{
			Amount:  utxo.Amount - amount,
			Address: privKey.Public().Address().Bytes(),
		})
	}

	sig := types.SignTransaction(&privKey, tx)
	tx.Inputs[0].Signature = sig.Bytes()

	_, err = c.HandleTransaction(ctx, tx)
	if err != nil {
		log.Fatal("failed to submit transaction:", err)
	}

	txHash := hex.EncodeToString(types.HashTransaction(tx))
	fmt.Println("Transaction submitted!")
	fmt.Println("TX Hash:", txHash)
	fmt.Printf("Sent %d to %s\n", amount, toAddrHex)
	if utxo.Amount > amount {
		fmt.Printf("Change: %d returned to sender\n", utxo.Amount-amount)
	}
}
