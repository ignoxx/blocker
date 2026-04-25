package node

import (
	"github.com/ignoxx/blocker/crypto"
)

type ValidatorSet struct {
	validators []*crypto.PublicKey
}

func NewValidatorSet(validators []*crypto.PublicKey) *ValidatorSet {
	return &ValidatorSet{
		validators: validators,
	}
}

func (s *ValidatorSet) ProposerIndex(height int32) int32 {
	return (height - 1) % int32(len(s.validators))
}

func (s *ValidatorSet) GetProposer(height int32) *crypto.PublicKey {
	return s.validators[s.ProposerIndex(height)]
}

func (s *ValidatorSet) Has(pubKey *crypto.PublicKey) bool {
	for _, v := range s.validators {
		if pubKey.String() == v.String() {
			return true
		}
	}
	return false
}

func (s *ValidatorSet) IndexOf(pubKey *crypto.PublicKey) int {
	for i, v := range s.validators {
		if pubKey.String() == v.String() {
			return i
		}
	}
	return -1
}

func (s *ValidatorSet) Len() int {
	return len(s.validators)
}
