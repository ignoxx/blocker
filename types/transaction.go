package types

import (
	"crypto/sha256"

	"github.com/ignoxx/blocker/crypto"
	"github.com/ignoxx/blocker/proto"
	pb "google.golang.org/protobuf/proto"
)

func SignTransaction(pk *crypto.PrivateKey, tx *proto.Transaction) crypto.Signature {
	return pk.Sign(HashTransaction(tx))
}

func HashTransaction(block *proto.Transaction) []byte {
	b, err := pb.Marshal(block)
	if err != nil {
		panic(err)
	}

	hash := sha256.Sum256(b)
	return hash[:]
}

func VerifyTransaction(tx *proto.Transaction) bool {
	for _, input := range tx.Inputs {
		sig := crypto.SignatureFromBytes(input.Signature)
		pubKey := crypto.PublicKeyFromBytes(input.PublicKey)

		// TODO: make sure we dont run into prob. after verification
		// because we do not want to hash the signature field
		input.Signature = nil
		if !sig.Verify(&pubKey, HashTransaction(tx)) {
			return false
		}
	}

	return true
}
