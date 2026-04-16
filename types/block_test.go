package types

import (
	"testing"

	"github.com/ignoxx/blocker/crypto"
	"github.com/ignoxx/blocker/proto"
	"github.com/ignoxx/blocker/util"
	"github.com/stretchr/testify/assert"
)

func TestCalculateRootHash(t *testing.T) {
	var (
		privKey, _ = crypto.GeneratePrivateKey()
		block      = util.RandomBlock(0, nil)
		tx         = &proto.Transaction{
			Version: 1,
		}
	)

	block.Transactions = append(block.Transactions, tx)
	SignBlock(privKey, block)

	assert.True(t, VerifyRootHash(block))
	assert.Equal(t, 32, len(block.Header.RootHash))
}

func TestSignVerifyBlock(t *testing.T) {
	var (
		block      = util.RandomBlock(0, nil)
		privKey, _ = crypto.GeneratePrivateKey()
		pubKey     = privKey.Public()
	)
	sig := SignBlock(privKey, block)
	assert.Len(t, sig.Bytes(), 64)
	assert.True(t, sig.Verify(pubKey, HashBlock(block)))

	assert.True(t, VerifyBlock(block))

	invalidPrivKey, _ := crypto.GeneratePrivateKey()
	block.PublicKey = invalidPrivKey.Public().Bytes()

	assert.False(t, VerifyBlock(block))
}

func TestHashBlock(t *testing.T) {
	block := util.RandomBlock(0, nil)
	hash := HashBlock(block)
	assert.Len(t, hash, 32)
}
