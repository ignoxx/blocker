package types

import (
	"testing"

	"github.com/ignoxx/blocker/crypto"
	"github.com/ignoxx/blocker/util"
	"github.com/stretchr/testify/assert"
)

func TestSignBlock(t *testing.T) {
	var (
		block      = util.RandomBlock()
		privKey, _ = crypto.GeneratePrivateKey()
		pubKey     = privKey.Public()
	)
	sig := SignBlock(privKey, block)
	assert.Len(t, sig.Bytes(), 64)
	assert.True(t, sig.Verify(pubKey, HashBlock(block)))
}

func TestHashBlock(t *testing.T) {
	block := util.RandomBlock()
	hash := HashBlock(block)
	assert.Len(t, hash, 32)
}
