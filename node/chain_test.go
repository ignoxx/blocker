package node

import (
	"encoding/hex"
	"testing"

	"github.com/ignoxx/blocker/crypto"
	"github.com/ignoxx/blocker/proto"
	"github.com/ignoxx/blocker/types"
	"github.com/ignoxx/blocker/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestAddBlockWithTx(t *testing.T) {
	privKey, _ := crypto.NewPrivateKeyFromSeedStr(MasterSeed)
	recipientPrivKey, _ := crypto.GeneratePrivateKey()
	recipient := recipientPrivKey.Public().Address()

	chain := NewChain(NewMemoryBlockStore(), NewMemoryTxStore())
	block := randomBlock(t, chain)

	ftx, err := chain.txStore.Get("240b46d4a4877ced7de4d298744aeef16141721cde0e8f762b2aff07e74689b4")
	assert.Nil(t, err)

	inputs := []*proto.TxInput{
		{
			PrevTxHash:   types.HashTransaction(ftx),
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

	block.Transactions = append(block.Transactions, tx)
	require.Nil(t, chain.AddBlock(block))
	txHash := hex.EncodeToString(types.HashTransaction(tx))

	fetchedTx, err := chain.txStore.Get(txHash)
	assert.Nil(t, err)
	assert.Equal(t, tx, fetchedTx)
}
