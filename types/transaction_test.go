package types

import (
	"testing"

	"github.com/ignoxx/blocker/crypto"
	"github.com/ignoxx/blocker/proto"
	"github.com/ignoxx/blocker/util"
	"github.com/stretchr/testify/assert"
)

// our init. balance: 100 coins
// we want to send 5 coins to "toAddress"
// 2 outputs:
// - 1. 5 coins to "toAddress"
// - 2. 95 coins to "fromAddress" aka. ourselves
func TestNewTransaction(t *testing.T) {
	var (
		fromPrivKey, _ = crypto.GeneratePrivateKey()
		fromPubKey     = fromPrivKey.Public()
		fromAddress    = fromPubKey.Address()

		toPrivKey, _ = crypto.GeneratePrivateKey()
		toPubKey     = toPrivKey.Public()
		toAddress    = toPubKey.Address()
	)

	input := &proto.TxInput{
		PrevTxHash:   util.RandomHash(),
		PrevOutIndex: 0,
		PublicKey:    fromPubKey.Bytes(),
	}

	output1 := &proto.TxOutput{
		Amount:  5,
		Address: toAddress.Bytes(),
	}

	output2 := &proto.TxOutput{
		Amount:  95,
		Address: fromAddress.Bytes(),
	}

	tx := &proto.Transaction{
		Version: 1,
		Inputs:  []*proto.TxInput{input},
		Outputs: []*proto.TxOutput{output1, output2},
	}

	sig := SignTransaction(&fromPrivKey, tx)
	input.Signature = sig.Bytes()

	assert.True(t, VerifyTransaction(tx))
}
