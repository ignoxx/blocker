package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

const (
	PublicKeySize  = 32
	PrivateKeySize = 64
	SeedSize       = 32
	AddressSize    = 20
	SigSize        = 64
)

type PrivateKey struct {
	key ed25519.PrivateKey
}

func GeneratePrivateKey() (PrivateKey, error) {
	seed := make([]byte, SeedSize)
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

func NewPrivateKeyFromSeedStr(seed string) (PrivateKey, error) {
	seedBytes, err := hex.DecodeString(seed)
	if err != nil {
		return PrivateKey{}, fmt.Errorf("failed to decode seed hex string: %w", err)
	}

	return NewPrivateKeyFromSeed(seedBytes)
}

func NewPrivateKeyFromSeed(seed []byte) (PrivateKey, error) {
	if len(seed) != SeedSize {
		return PrivateKey{}, fmt.Errorf("invalid seed length: expected %d bytes, got %d bytes", SeedSize, len(seed))
	}

	return PrivateKey{
		key: ed25519.NewKeyFromSeed(seed),
	}, nil
}

func (p PrivateKey) String() string {
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

func (p *PrivateKey) Public() *PublicKey {
	b := make([]byte, PublicKeySize)
	copy(b, p.key[len(p.key)-PublicKeySize:])
	return &PublicKey{
		key: b,
	}
}

type PublicKey struct {
	key ed25519.PublicKey
}

func PublicKeyFromBytes(b []byte) PublicKey {
	if len(b) != PublicKeySize {
		panic(fmt.Sprintf("invalid public key length: expected %d bytes, got %d bytes", PublicKeySize, len(b)))
	}

	return PublicKey{
		key: b,
	}
}

func (p *PublicKey) Address() *Address {
	return &Address{
		value: p.key[len(p.key)-AddressSize:],
	}
}

func (p PublicKey) String() string {
	return hex.EncodeToString(p.key)
}

func (p *PublicKey) Bytes() []byte {
	return p.key
}

type Signature struct {
	value []byte
}

func SignatureFromBytes(b []byte) Signature {
	if len(b) != SigSize {
		panic(fmt.Sprintf("invalid signature length: expected %d bytes, got %d bytes", SigSize, len(b)))
	}

	return Signature{
		value: b,
	}
}
func (s *Signature) Bytes() []byte {
	return s.value
}

func (s *Signature) Verify(pubKey *PublicKey, msg []byte) bool {
	return ed25519.Verify(pubKey.key, msg, s.value)
}

type Address struct {
	value []byte
}

func AddressFromBytes(b []byte) Address {
	if len(b) != AddressSize {
		panic(fmt.Sprintf("invalid address length: expected %d bytes, got %d bytes", AddressSize, len(b)))
	}

	return Address{
		value: b,
	}
}

func (a Address) String() string {
	return hex.EncodeToString(a.value)
}

func (a *Address) Bytes() []byte {
	return a.value
}
