package node

import (
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/ignoxx/blocker/proto"
	"github.com/ignoxx/blocker/types"
)

var _ BlockStorer = (*MemoryBlockStore)(nil)

type BlockStorer interface {
	Put(*proto.Block) error
	Get(string) (*proto.Block, error)
}

type MemoryBlockStore struct {
	lock   sync.RWMutex
	blocks map[string]*proto.Block
}

func NewMemoryBlockStore() *MemoryBlockStore {
	return &MemoryBlockStore{
		blocks: map[string]*proto.Block{},
	}
}

func (m *MemoryBlockStore) Get(hash string) (*proto.Block, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	block, ok := m.blocks[hash]
	if !ok {
		return nil, fmt.Errorf("block not found: %s", hash)
	}

	return block, nil
}

func (m *MemoryBlockStore) Put(block *proto.Block) error {
	m.lock.RLock()
	defer m.lock.RUnlock()
	hash := hex.EncodeToString(types.HashBlock(block))
	m.blocks[hash] = block
	return nil
}
