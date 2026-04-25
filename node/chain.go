package node

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/ignoxx/blocker/crypto"
	"github.com/ignoxx/blocker/proto"
	"github.com/ignoxx/blocker/types"
)

var MasterSeed = "0f71b4d49b833860f40ef07c39556287a239c1a27bc26fd5261b99f56af867f2"

type HeaderList struct {
	headers []*proto.Header
}

func NewHeaderList() *HeaderList {
	return &HeaderList{
		headers: []*proto.Header{},
	}
}

func (list *HeaderList) Add(h *proto.Header) {
	list.headers = append(list.headers, h)
}

func (list *HeaderList) Get(index int) *proto.Header {
	if index > list.Height() {
		panic(fmt.Sprintf("index out of bounds: %d", index))
	}

	return list.headers[index]
}

func (list *HeaderList) Len() int {
	return len(list.headers)
}

func (list *HeaderList) Height() int {
	return len(list.headers) - 1
}

type UTXO struct {
	Hash   string
	Index  int
	Amount int64
	Spent  bool
}

type Chain struct {
	txStore    TxStorer
	blockStore BlockStorer
	utxoStore  UTXOStorer
	headers    *HeaderList
	validators *ValidatorSet
}

func NewChain(bs BlockStorer, txStore TxStorer, validators *ValidatorSet) *Chain {
	chain := &Chain{
		txStore:    txStore,
		blockStore: bs,
		utxoStore:  NewUTXOStore(),
		headers:    NewHeaderList(),
		validators: validators,
	}

	chain.addBlock(createGenesisBlock())
	return chain
}

func (c *Chain) Height() int {
	return c.headers.Height()
}

func (c *Chain) AddBlock(b *proto.Block) error {
	if err := c.ValidateBlock(b); err != nil {
		return err
	}

	return c.addBlock(b)
}

func (c *Chain) addBlock(b *proto.Block) error {
	c.headers.Add(b.Header)

	for _, tx := range b.Transactions {
		fmt.Println("NEW TX", hex.EncodeToString(types.HashTransaction(tx)))
		if err := c.txStore.Put(tx); err != nil {
			return fmt.Errorf("failed to store transaction: %w", err)
		}

		hash := hex.EncodeToString(types.HashTransaction(tx))
		for i, output := range tx.Outputs {
			utxo := &UTXO{
				Hash:   hash,
				Index:  i,
				Amount: output.Amount,
				Spent:  false,
			}

			if err := c.utxoStore.Put(utxo); err != nil {
				return fmt.Errorf("failed to store utxo: %w", err)
			}
		}

		for _, input := range tx.Inputs {
			key := fmt.Sprintf("%s:%d", hex.EncodeToString(input.PrevTxHash), input.PrevOutIndex)
			utxo, err := c.utxoStore.Get(key)
			if err != nil {
				return fmt.Errorf("failed to get utxo: %w", err)
			}

			utxo.Spent = true
			if err := c.utxoStore.Put(utxo); err != nil {
				return fmt.Errorf("failed to update utxo: %w", err)
			}
		}
	}

	return c.blockStore.Put(b)
}

func (c *Chain) GetBlockByHash(hash []byte) (*proto.Block, error) {
	hashHex := hex.EncodeToString(hash)
	return c.blockStore.Get(hashHex)
}

func (c *Chain) GetBlockByHeight(height int) (*proto.Block, error) {
	if c.Height() < height {
		return nil, fmt.Errorf("height out of bounds: %d", height)
	}

	header := c.headers.Get(height)
	hash := types.HashHeader(header)
	return c.GetBlockByHash(hash)
}

func (c *Chain) ValidateBlock(b *proto.Block) error {
	if !types.VerifyBlock(b) {
		return fmt.Errorf("invalid block: signature verification failed")
	}

	expectedHeight := c.Height() + 1
	if b.Header.Height != int32(expectedHeight) {
		return fmt.Errorf("invalid block: expected height %d, got %d", expectedHeight, b.Header.Height)
	}

	signerPubKey := crypto.PublicKeyFromBytes(b.PublicKey)
	if !c.validators.Has(&signerPubKey) {
		return fmt.Errorf("invalid block: signer is not a validator")
	}

	expectedProposer := c.validators.GetProposer(b.Header.Height)
	if signerPubKey.String() != expectedProposer.String() {
		return fmt.Errorf("invalid block: wrong proposer at height %d", b.Header.Height)
	}

	currentBlock, err := c.GetBlockByHeight(c.Height())
	if err != nil {
		return err
	}

	hash := types.HashBlock(currentBlock)
	if !bytes.Equal(hash, b.Header.PrevHash) {
		return fmt.Errorf("invalid block: expected prev hash %s, got %s", hex.EncodeToString(hash), hex.EncodeToString(b.Header.PrevHash))
	}

	for _, tx := range b.Transactions {
		if err := c.ValidateTransaction(tx); err != nil {
			return fmt.Errorf("invalid block: %w", err)
		}
	}

	return nil
}

func (c *Chain) ValidateTransaction(tx *proto.Transaction) error {
	if !types.VerifyTransaction(tx) {
		return fmt.Errorf("invalid block: transaction signature verification failed")
	}

	sumInputs := 0

	for i := range tx.Inputs {
		key := fmt.Sprintf("%s:%d", hex.EncodeToString(tx.Inputs[i].PrevTxHash), tx.Inputs[i].PrevOutIndex)
		utxo, err := c.utxoStore.Get(key)
		if err != nil {
			return fmt.Errorf("invalid block: %w", err)
		}
		sumInputs += int(utxo.Amount)

		if utxo.Spent {
			return fmt.Errorf("invalid block: transaction input %d is already spent", i)
		}
	}

	sumOutputs := 0
	for _, output := range tx.Outputs {
		sumOutputs += int(output.Amount)
	}

	if sumInputs < sumOutputs {
		return fmt.Errorf("invalid block: transaction inputs %d are less than outputs %d", sumInputs, sumOutputs)
	}

	return nil
}

func createGenesisBlock() *proto.Block {
	privKey, _ := crypto.NewPrivateKeyFromSeedStr(MasterSeed)
	block := &proto.Block{
		Header: &proto.Header{
			Version:   1,
			Height:    0,
			Timestamp: 0,
		},
	}

	tx := &proto.Transaction{
		Version: 1,
		Inputs:  []*proto.TxInput{},
		Outputs: []*proto.TxOutput{
			{
				Amount:  1_000,
				Address: privKey.Public().Address().Bytes(),
			},
		},
	}

	block.Transactions = append(block.Transactions, tx)
	types.SignBlock(privKey, block)

	return block
}
