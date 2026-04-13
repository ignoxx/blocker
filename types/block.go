package types

import (
	"crypto/sha256"

	"github.com/ignoxx/blocker/crypto"
	"github.com/ignoxx/blocker/proto"
	pb "google.golang.org/protobuf/proto"
)

func VerifyBlock(block *proto.Block) bool {
	if len(block.PublicKey) != crypto.PublicKeySize {
		return false
	}

	if len(block.Signature) != crypto.SigSize {
		return false
	}

	sig := crypto.SignatureFromBytes(block.Signature)
	pubKey := crypto.PublicKeyFromBytes(block.PublicKey)
	return sig.Verify(&pubKey, HashBlock(block))
}

func SignBlock(pk crypto.PrivateKey, block *proto.Block) crypto.Signature {
	hash := HashBlock(block)
	sig := pk.Sign(hash)
	block.PublicKey = pk.Public().Bytes()
	block.Signature = sig.Bytes()
	return sig
}

func HashBlock(block *proto.Block) []byte {
	return HashHeader(block.Header)
}

func HashHeader(header *proto.Header) []byte {
	b, err := pb.Marshal(header)
	if err != nil {
		panic(err)
	}

	hash := sha256.Sum256(b)
	return hash[:]
}
