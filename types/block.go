package types

import (
	"bytes"
	"crypto/sha256"

	"github.com/cbergoon/merkletree"
	"github.com/ignoxx/blocker/crypto"
	"github.com/ignoxx/blocker/proto"
	pb "google.golang.org/protobuf/proto"
)

type TxHash struct {
	hash []byte
}

func NewTxHash(hash []byte) TxHash {
	return TxHash{
		hash: hash,
	}
}

func (t TxHash) CalculateHash() ([]byte, error) {
	return t.hash, nil
}

func (t TxHash) Equals(other merkletree.Content) (bool, error) {
	equals := bytes.Equal(t.hash, other.(TxHash).hash)
	return equals, nil
}

func VerifyBlock(block *proto.Block) bool {
	if len(block.Transactions) > 0 {
		if !VerifyRootHash(block) {
			return false
		}
	}

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
	if len(block.Transactions) > 0 {
		tree, err := GetMerkleTree(block)
		if err != nil {
			panic(err)
		}

		block.Header.RootHash = tree.MerkleRoot()
	}

	hash := HashBlock(block)
	sig := pk.Sign(hash)
	block.PublicKey = pk.Public().Bytes()
	block.Signature = sig.Bytes()

	return sig
}

func VerifyRootHash(b *proto.Block) bool {
	tree, err := GetMerkleTree(b)
	if err != nil {
		return false
	}

	valid, err := tree.VerifyTree()
	if err != nil {
		return false
	}

	return valid
}

func GetMerkleTree(b *proto.Block) (*merkletree.MerkleTree, error) {
	list := make([]merkletree.Content, len(b.Transactions))
	for i := range b.Transactions {
		list[i] = NewTxHash(HashTransaction(b.Transactions[i]))
	}

	t, err := merkletree.NewTree(list)
	if err != nil {
		return nil, err
	}

	return t, nil
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
