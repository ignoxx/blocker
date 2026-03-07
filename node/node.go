package node

import (
	"context"
	"log/slog"

	"github.com/ignoxx/blocker/proto"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	version = "blocker-v0.1"
)

type Node struct {
	version string
	proto.UnimplementedNodeServer
}

func New() *Node {
	return &Node{
		version: version,
	}
}

func (n *Node) HandleTransaction(ctx context.Context, tx *proto.Transaction) (*emptypb.Empty, error) {
	slog.Info("Received transaction", "transaction", tx)
	return nil, nil
}

func (n *Node) Handshake(ctx context.Context, v *proto.Version) (*proto.Version, error) {
	peer, _ := peer.FromContext(ctx)
	slog.Info("Received handshake", "from", peer.Addr, "version", v)
	return &proto.Version{
		Version: n.version,
		Height:  100,
	}, nil
}

// type Server struct {
// 	lnAddr string
// 	ln     net.Listener
// }
//
// func New(lnAddr string) (*Server, error) {
// 	ln, err := net.Listen("tcp", lnAddr)
// 	if err != nil {
// 		return nil, fmt.Errorf("new server: %s: %w", lnAddr, err)
// 	}
//
// 	return &Server{
// 		lnAddr: lnAddr,
// 		ln:     ln,
// 	}, nil
// }
