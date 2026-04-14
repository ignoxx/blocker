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

// len = count of headers in the list
// height = index of the header in the list
func (list *HeaderList) Len() int {
	return len(list.headers)
}

// there will be always be a header at height 0, which is the genesis header/block
func (list *HeaderList) Height() int {
	return len(list.headers) - 1
}

type Chain struct {
	txStore    TxStorer
	blockStore BlockStorer
	headers    *HeaderList
}

func NewChain(bs BlockStorer, txStore TxStorer) *Chain {
	chain := &Chain{
		txStore:    txStore,
		blockStore: bs,
		headers:    NewHeaderList(),
	}

	// TODO: just for testing
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
	// validate the signature of the block
	if !types.VerifyBlock(b) {
		return fmt.Errorf("invalid block: signature verification failed")
	}

	// validate if the prevHash is the actually hash of the current block
	currentBlock, err := c.GetBlockByHeight(c.Height())
	if err != nil {
		return err
	}

	hash := types.HashBlock(currentBlock)
	if !bytes.Equal(hash, b.Header.PrevHash) {
		return fmt.Errorf("invalid block: expected prev hash %s, got %s", hex.EncodeToString(hash), hex.EncodeToString(b.Header.PrevHash))
	}

	return nil
}

func createGenesisBlock() *proto.Block {
	privKey, _ := crypto.NewPrivateKeyFromSeedStr(MasterSeed)
	block := &proto.Block{
		Header: &proto.Header{
			Version: 1,
		},
	}

	tx := &proto.Transaction{
		Version: 1,
		Inputs:  []*proto.TxInput{},
		Outputs: []*proto.TxOutput{
			{
				Amount:  1_000,
				Address: privKey.Public().Bytes(),
			},
		},
	}

	block.Transactions = append(block.Transactions, tx)
	types.SignBlock(privKey, block)

	return block
}
