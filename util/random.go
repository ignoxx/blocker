package util

import (
	randc "crypto/rand"
	"io"
	"time"

	"github.com/ignoxx/blocker/proto"
)

func RandomHash() []byte {
	hash := make([]byte, 32)
	io.ReadFull(randc.Reader, hash)
	return hash
}

func RandomBlock(height int, prevHash []byte) *proto.Block {
	header := &proto.Header{
		Version:   1,
		Height:    int32(height),
		PrevHash:  prevHash,
		Timestamp: time.Now().UnixNano(),
	}

	return &proto.Block{
		Header: header,
	}
}
