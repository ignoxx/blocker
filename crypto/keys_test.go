package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenPrivateKey(t *testing.T) {
	privKey, err := GeneratePrivateKey()
	assert.NoError(t, err)
	assert.Equal(t, PrivateKeySize, len(privKey.Bytes()))

	pubKey := privKey.Public()
	assert.Equal(t, PublicKeySize, len(pubKey.Bytes()))
}

func TestPrivateKeySign(t *testing.T) {
	var (
		privKey, _ = GeneratePrivateKey()
		pubKey     = privKey.Public()
		msg        = []byte("some super random foo bar baz")
	)
	sig := privKey.Sign(msg)
	assert.True(t, sig.Verify(pubKey, msg))

	// test with invalid msg
	assert.False(t, sig.Verify(pubKey, []byte("ur mom")))

	// test with invalid pubKey
	invalidPrivKey, _ := GeneratePrivateKey()
	invalidPubKey := invalidPrivKey.Public()
	assert.False(t, sig.Verify(invalidPubKey, msg))
}

func TestPublicKeyToAddress(t *testing.T) {
	privKey, _ := GeneratePrivateKey()
	pubKey := privKey.Public()
	addr := pubKey.Address()

	assert.Equal(t, AddressSize, len(addr.Bytes()))
}

func TestNewPrivateKeyFromString(t *testing.T) {
	var (
		seedHx       = "4ed6fdd83148139a38a99fa2ca58561600c70444a9a4972105c34a300fa385f6"
		address      = "d9b560f4d76fd08a818b84ed653f3d0ae4ee4d0d"
		privKey, err = NewPrivateKeyFromString(seedHx)
	)
	assert.NoError(t, err)
	assert.Len(t, privKey.Bytes(), PrivateKeySize)

	var (
		actualpubKey = privKey.Public()
		addr         = actualpubKey.Address()
	)
	assert.NotEmpty(t, addr)
	assert.Len(t, addr.Bytes(), AddressSize)
	assert.Equal(t, address, addr.String())
}
