package node

import (
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/ignoxx/blocker/proto"
	"github.com/ignoxx/blocker/types"
)

var _ BlockStorer = (*MemoryBlockStore)(nil)

type UTXOStorer interface {
	Put(*UTXO) error
	Get(string) (*UTXO, error)
}

type UTXOStore struct {
	lock sync.RWMutex
	data map[string]*UTXO
}

func NewUTXOStore() *UTXOStore {
	return &UTXOStore{
		data: map[string]*UTXO{},
	}
}

func (s *UTXOStore) Put(utxo *UTXO) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	key := fmt.Sprintf("%s:%d", utxo.Hash, utxo.Index)
	s.data[key] = utxo
	return nil
}

func (s *UTXOStore) Get(key string) (*UTXO, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	utxo, ok := s.data[key]
	if !ok {
		return nil, fmt.Errorf("utxo not found: %s", key)
	}

	return utxo, nil
}

type TxStorer interface {
	Put(*proto.Transaction) error
	Get(string) (*proto.Transaction, error)
}

type MemoryTxStore struct {
	lock sync.RWMutex
	txx  map[string]*proto.Transaction
}

func NewMemoryTxStore() *MemoryTxStore {
	return &MemoryTxStore{
		txx: map[string]*proto.Transaction{},
	}
}

func (s *MemoryTxStore) Put(tx *proto.Transaction) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	hash := hex.EncodeToString(types.HashTransaction(tx))
	s.txx[hash] = tx
	return nil
}

func (s *MemoryTxStore) Get(hash string) (*proto.Transaction, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	tx, ok := s.txx[hash]
	if !ok {
		return nil, fmt.Errorf("transaction not found: %s", hash)
	}

	return tx, nil
}

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
	m.lock.Lock()
	defer m.lock.Unlock()
	hash := hex.EncodeToString(types.HashBlock(block))
	m.blocks[hash] = block
	return nil
}
