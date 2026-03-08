package node

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/ignoxx/blocker/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	version = "blocker-v0.1"
)

type Node struct {
	lnAddr  string
	height  int32
	version string
	logger  *slog.Logger

	peerLock sync.RWMutex
	peers    map[proto.NodeClient]*proto.Version

	proto.UnimplementedNodeServer
}

func New() *Node {
	return &Node{
		version: version,
		logger:  slog.Default(),
		peers:   make(map[proto.NodeClient]*proto.Version),
	}
}

func (n *Node) addPeer(c proto.NodeClient, v *proto.Version) {
	n.peerLock.Lock()
	defer n.peerLock.Unlock()

	n.log("new peer connected", "height", v.Height, "addr", v.ListenAddr)

	n.peers[c] = v
}

func (n *Node) deletePeer(c proto.NodeClient) {
	n.peerLock.Lock()
	defer n.peerLock.Unlock()
	delete(n.peers, c)
}

func (n *Node) Start(lnAddr string) error {
	n.lnAddr = lnAddr

	grpcServer := grpc.NewServer()
	ln, err := net.Listen("tcp", lnAddr)
	if err != nil {
		return errors.New("node start: " + err.Error())
	}

	proto.RegisterNodeServer(grpcServer, n)
	n.log("Starting gRPC server", "address", lnAddr)

	return grpcServer.Serve(ln)
}

func (n *Node) HandleTransaction(ctx context.Context, tx *proto.Transaction) (*emptypb.Empty, error) {
	n.log("received transaction", "transaction", tx)
	return nil, nil
}

func (n *Node) Handshake(ctx context.Context, v *proto.Version) (*proto.Version, error) {
	peer, _ := peer.FromContext(ctx)
	_, err := makeNodeClient(v.ListenAddr)
	if err != nil {
		return nil, errors.New("handshake: " + err.Error())
	}

	n.log("received handshake", "version", v.Version, "addr", v.ListenAddr, "peerAddr", peer.Addr, "height", v.Height)

	return n.getVersion(), nil
}

func (n *Node) getVersion() *proto.Version {
	return &proto.Version{
		Version:    n.version,
		Height:     n.height,
		ListenAddr: n.lnAddr,
	}
}

func (n *Node) BootstrapNetwork(addrs []string) error {
	for _, addr := range addrs {
		c, err := makeNodeClient(addr)
		if err != nil {
			return errors.New("bootstrap network: " + err.Error())
		}

		v, err := c.Handshake(context.TODO(), n.getVersion())
		if err != nil {
			return errors.New("handshake: " + err.Error())
		}

		n.addPeer(c, v)
	}

	return nil
}

func (n *Node) log(msg string, args ...any) {
	n.logger.Info(
		fmt.Sprintf("[%s] %s", n.lnAddr, msg),
		args...,
	)
}

func makeNodeClient(lnAddr string) (proto.NodeClient, error) {
	client, err := grpc.NewClient(lnAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, errors.New("make node client: " + err.Error())
	}

	return proto.NewNodeClient(client), nil
}
