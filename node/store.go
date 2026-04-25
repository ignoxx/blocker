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
	List() []*UTXO
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

func (s *UTXOStore) List() []*UTXO {
	s.lock.RLock()
	defer s.lock.RUnlock()

	list := make([]*UTXO, 0, len(s.data))
	for _, v := range s.data {
		list = append(list, v)
	}
	return list
}

type TxStorer interface {
	Put(*proto.Transaction) error
	Get(string) (*proto.Transaction, error)
	List() []*proto.Transaction
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

func (s *MemoryTxStore) List() []*proto.Transaction {
	s.lock.RLock()
	defer s.lock.RUnlock()

	list := make([]*proto.Transaction, 0, len(s.txx))
	for _, v := range s.txx {
		list = append(list, v)
	}
	return list
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
