package node

import (
	"encoding/hex"
	"fmt"

	"github.com/ignoxx/blocker/proto"
	"github.com/ignoxx/blocker/types"
)

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
	blockStore BlockStorer
	headers    *HeaderList
}

func NewChain(bs BlockStorer) *Chain {
	return &Chain{
		blockStore: bs,
		headers:    NewHeaderList(),
	}
}

func (c *Chain) Height() int {
	return c.headers.Height()
}

func (c *Chain) AddBlock(b *proto.Block) error {
	c.headers.Add(b.Header)
	// TODO: validation
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
