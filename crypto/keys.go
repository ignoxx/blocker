package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

const (
	publicKeySize  = 32
	privateKeySize = 64
	seedSize       = 32
	addressSize    = 20
)

type PrivateKey struct {
	key ed25519.PrivateKey
}

func GeneratePrivateKey() (PrivateKey, error) {
	seed := make([]byte, seedSize)
	_, err := io.ReadFull(rand.Reader, seed)
	if err != nil {
		return PrivateKey{}, fmt.Errorf("failed to generate random seed: %w", err)
	}

	return PrivateKey{
		key: ed25519.NewKeyFromSeed(seed),
	}, nil
}

func NewPrivateKeyFromString(seedHx string) (PrivateKey, error) {
	seed, err := hex.DecodeString(seedHx)
	if err != nil {
		return PrivateKey{}, fmt.Errorf("failed to decode seed hex string: %w", err)
	}

	return NewPrivateKeyFromSeed(seed)
}

func NewPrivateKeyFromSeed(seed []byte) (PrivateKey, error) {
	if len(seed) != seedSize {
		return PrivateKey{}, fmt.Errorf("invalid seed length: expected %d bytes, got %d bytes", seedSize, len(seed))
	}

	return PrivateKey{
		key: ed25519.NewKeyFromSeed(seed),
	}, nil
}

func (p *PrivateKey) String() string {
	return hex.EncodeToString(p.key)
}

func (p *PrivateKey) Bytes() []byte {
	return p.key
}

func (p *PrivateKey) Sign(msg []byte) Signature {
	return Signature{
		value: ed25519.Sign(p.key, msg),
	}
}

func (p *PrivateKey) Public() PublicKey {
	b := make([]byte, publicKeySize)
	copy(b, p.key[len(p.key)-publicKeySize:])
	return PublicKey{
		key: b,
	}
}

type PublicKey struct {
	key ed25519.PublicKey
}

func (p *PublicKey) Address() Address {
	return Address{
		value: p.key[len(p.key)-addressSize:],
	}
}

func (p *PublicKey) String() string {
	return hex.EncodeToString(p.key)
}

func (p *PublicKey) Bytes() []byte {
	return p.key
}

type Signature struct {
	value []byte
}

func (s *Signature) Bytes() []byte {
	return s.value
}

func (s *Signature) Verify(pubKey PublicKey, msg []byte) bool {
	return ed25519.Verify(pubKey.key, msg, s.value)
}

type Address struct {
	value []byte
}

func (a *Address) String() string {
	return hex.EncodeToString(a.value)
}

func (a *Address) Bytes() []byte {
	return a.value
}
