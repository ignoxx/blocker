package node

import (
	"testing"

	"github.com/ignoxx/blocker/crypto"
	"github.com/ignoxx/blocker/proto"
	"github.com/ignoxx/blocker/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatorSetRoundRobin(t *testing.T) {
	privKey1, _ := crypto.GeneratePrivateKey()
	privKey2, _ := crypto.GeneratePrivateKey()
	privKey3, _ := crypto.GeneratePrivateKey()

	validators := NewValidatorSet([]*crypto.PublicKey{
		privKey1.Public(),
		privKey2.Public(),
		privKey3.Public(),
	})

	assert.Equal(t, 3, validators.Len())

	assert.Equal(t, privKey1.Public().String(), validators.GetProposer(1).String())
	assert.Equal(t, privKey2.Public().String(), validators.GetProposer(2).String())
	assert.Equal(t, privKey3.Public().String(), validators.GetProposer(3).String())
	assert.Equal(t, privKey1.Public().String(), validators.GetProposer(4).String())
	assert.Equal(t, privKey2.Public().String(), validators.GetProposer(5).String())
	assert.Equal(t, privKey3.Public().String(), validators.GetProposer(6).String())
}

func TestValidatorSetHas(t *testing.T) {
	privKey1, _ := crypto.GeneratePrivateKey()
	privKey2, _ := crypto.GeneratePrivateKey()

	validators := NewValidatorSet([]*crypto.PublicKey{
		privKey1.Public(),
	})

	assert.True(t, validators.Has(privKey1.Public()))
	assert.False(t, validators.Has(privKey2.Public()))
}

func TestBlockProposerValidation(t *testing.T) {
	privKey1, _ := crypto.GeneratePrivateKey()
	privKey2, _ := crypto.GeneratePrivateKey()
	privKey3, _ := crypto.GeneratePrivateKey()

	validators := NewValidatorSet([]*crypto.PublicKey{
		privKey1.Public(),
		privKey2.Public(),
		privKey3.Public(),
	})

	chain := NewChain(NewMemoryBlockStore(), NewMemoryTxStore(), validators)
	require.Equal(t, 0, chain.Height())

	genesisBlock, err := chain.GetBlockByHeight(0)
	require.NoError(t, err)
	prevHash := types.HashBlock(genesisBlock)

	block := &proto.Block{
		Header: &proto.Header{
			Version:  1,
			Height:   1,
			PrevHash: prevHash,
		},
	}

	types.SignBlock(privKey1, block)

	err = chain.AddBlock(block)
	assert.NoError(t, err)
	assert.Equal(t, 1, chain.Height())
}

func TestBlockWrongHeightFails(t *testing.T) {
	privKey1, _ := crypto.GeneratePrivateKey()
	privKey2, _ := crypto.GeneratePrivateKey()
	privKey3, _ := crypto.GeneratePrivateKey()

	validators := NewValidatorSet([]*crypto.PublicKey{
		privKey1.Public(),
		privKey2.Public(),
		privKey3.Public(),
	})

	chain := NewChain(NewMemoryBlockStore(), NewMemoryTxStore(), validators)

	genesisBlock, _ := chain.GetBlockByHeight(0)
	prevHash := types.HashBlock(genesisBlock)

	block := &proto.Block{
		Header: &proto.Header{
			Version:  1,
			Height:   5,
			PrevHash: prevHash,
		},
	}

	types.SignBlock(privKey1, block)
	err := chain.AddBlock(block)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected height 1")
}

func TestBlockWrongProposerFails(t *testing.T) {
	privKey1, _ := crypto.GeneratePrivateKey()
	privKey2, _ := crypto.GeneratePrivateKey()
	privKey3, _ := crypto.GeneratePrivateKey()

	validators := NewValidatorSet([]*crypto.PublicKey{
		privKey1.Public(),
		privKey2.Public(),
		privKey3.Public(),
	})

	chain := NewChain(NewMemoryBlockStore(), NewMemoryTxStore(), validators)

	genesisBlock, _ := chain.GetBlockByHeight(0)
	prevHash := types.HashBlock(genesisBlock)

	block := &proto.Block{
		Header: &proto.Header{
			Version:  1,
			Height:   1,
			PrevHash: prevHash,
		},
	}

	types.SignBlock(privKey2, block)
	err := chain.AddBlock(block)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wrong proposer")
}
