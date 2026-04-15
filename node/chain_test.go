package node

import (
	"testing"

	"github.com/ignoxx/blocker/crypto"
	"github.com/ignoxx/blocker/proto"
	"github.com/ignoxx/blocker/types"
	"github.com/ignoxx/blocker/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const genesisTxHash = "a57f3604f7ef10fb992523eb9b9ecf2022a462b1caea9b5bc74207a66fd28b95"

func randomBlock(t *testing.T, chain *Chain) *proto.Block {
	privKey, _ := crypto.GeneratePrivateKey()
	block := util.RandomBlock()
	prevBlock, err := chain.GetBlockByHeight(chain.Height())
	require.Nil(t, err)
	block.Header.PrevHash = types.HashBlock(prevBlock)
	types.SignBlock(privKey, block)
	return block
}

func TestChainHeight(t *testing.T) {
	chain := NewChain(NewMemoryBlockStore(), NewMemoryTxStore())
	for i := range 100 {
		block := randomBlock(t, chain)
		require.Nil(t, chain.AddBlock(block))
		require.Equal(t, i+1, chain.Height())
	}
}

func TestAddBlock(t *testing.T) {
	chain := NewChain(NewMemoryBlockStore(), NewMemoryTxStore())

	for i := range 100 {
		block := randomBlock(t, chain)
		blockHash := types.HashBlock(block)

		require.Nil(t, chain.AddBlock(block))
		fetchedBlock, err := chain.GetBlockByHash(blockHash)
		require.Nil(t, err)
		require.Equal(t, block, fetchedBlock)

		fetchedBlockByHeight, err := chain.GetBlockByHeight(i + 1)
		require.Nil(t, err)
		require.Equal(t, block, fetchedBlockByHeight)
	}
}

func TestAddBlockWithTxLowFunds(t *testing.T) {
	privKey, _ := crypto.NewPrivateKeyFromSeedStr(MasterSeed)
	recipientPrivKey, _ := crypto.GeneratePrivateKey()
	recipient := recipientPrivKey.Public().Address()

	chain := NewChain(NewMemoryBlockStore(), NewMemoryTxStore())
	block := randomBlock(t, chain)

	prevTx, err := chain.txStore.Get(genesisTxHash)
	assert.Nil(t, err)

	inputs := []*proto.TxInput{
		{
			PrevTxHash:   types.HashTransaction(prevTx),
			PrevOutIndex: 0,
			PublicKey:    privKey.Public().Bytes(),
		},
	}

	outputs := []*proto.TxOutput{
		{
			Amount:  1001,
			Address: recipient.Bytes(),
		},
	}

	tx := &proto.Transaction{
		Version: 1,
		Inputs:  inputs,
		Outputs: outputs,
	}

	sig := types.SignTransaction(&privKey, tx)
	tx.Inputs[0].Signature = sig.Bytes()

	block.Transactions = append(block.Transactions, tx)
	require.NotNil(t, chain.AddBlock(block))

}

func TestAddBlockWithTx(t *testing.T) {
	privKey, _ := crypto.NewPrivateKeyFromSeedStr(MasterSeed)
	recipientPrivKey, _ := crypto.GeneratePrivateKey()
	recipient := recipientPrivKey.Public().Address()

	chain := NewChain(NewMemoryBlockStore(), NewMemoryTxStore())
	block := randomBlock(t, chain)

	prevTx, err := chain.txStore.Get(genesisTxHash)
	assert.Nil(t, err)

	inputs := []*proto.TxInput{
		{
			PrevTxHash:   types.HashTransaction(prevTx),
			PrevOutIndex: 0,
			PublicKey:    privKey.Public().Bytes(),
		},
	}

	outputs := []*proto.TxOutput{
		{
			Amount:  100,
			Address: recipient.Bytes(),
		},
		{
			Amount:  900,
			Address: privKey.Public().Address().Bytes(),
		},
	}

	tx := &proto.Transaction{
		Version: 1,
		Inputs:  inputs,
		Outputs: outputs,
	}

	sig := types.SignTransaction(&privKey, tx)
	tx.Inputs[0].Signature = sig.Bytes()

	block.Transactions = append(block.Transactions, tx)
	require.Nil(t, chain.AddBlock(block))
}
