package node

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"slices"
	"sync"
	"time"

	"github.com/ignoxx/blocker/crypto"
	"github.com/ignoxx/blocker/proto"
	"github.com/ignoxx/blocker/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	blockTime = time.Second * 5
)

type Mempool struct {
	lock sync.RWMutex
	txx  map[string]*proto.Transaction
}

func NewMempool() *Mempool {
	return &Mempool{
		txx: map[string]*proto.Transaction{},
	}
}

func (pool *Mempool) Has(tx *proto.Transaction) bool {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	hash := hex.EncodeToString(types.HashTransaction(tx))
	_, ok := pool.txx[hash]
	return ok
}

func (pool *Mempool) Add(tx *proto.Transaction) bool {
	if pool.Has(tx) {
		return false
	}

	pool.lock.Lock()
	defer pool.lock.Unlock()

	hash := hex.EncodeToString(types.HashTransaction(tx))
	pool.txx[hash] = tx
	return true
}

func (pool *Mempool) Len() int {
	pool.lock.RLock()
	defer pool.lock.RUnlock()

	return len(pool.txx)
}

func (pool *Mempool) Clear() []*proto.Transaction {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	txx := make([]*proto.Transaction, len(pool.txx))
	it := 0
	for k, v := range pool.txx {
		delete(pool.txx, k)
		txx[it] = v
		it++
	}

	return txx
}

type ServerConfig struct {
	Version    string
	ListenAddr string
	PrivateKey *crypto.PrivateKey
}

type Node struct {
	ServerConfig
	lnAddr  string
	height  int32
	version string
	logger  *slog.Logger

	peerLock sync.RWMutex
	peers    map[proto.NodeClient]*proto.Version
	mempool  *Mempool
	// TODO: might need a mutex for mempool too

	proto.UnimplementedNodeServer
}

func NewNode(cfg ServerConfig) *Node {
	return &Node{
		logger:       slog.Default(),
		peers:        make(map[proto.NodeClient]*proto.Version),
		mempool:      NewMempool(),
		ServerConfig: cfg,
	}
}

func (n *Node) addPeer(c proto.NodeClient, v *proto.Version) {
	n.peerLock.Lock()
	defer n.peerLock.Unlock()

	// TODO: handle the logic where we decide if we accept/drop the con
	// ..

	n.peers[c] = v

	if len(v.PeerList) > 0 {
		go n.bootstrapNetwork(v.PeerList)
	}

	n.log("new peer successfuly connected", "height", v.Height, "addr", v.ListenAddr)
}

func (n *Node) deletePeer(c proto.NodeClient) {
	n.peerLock.Lock()
	defer n.peerLock.Unlock()
	delete(n.peers, c)
}

func (n *Node) Start(bootstrapNodes []string) error {
	grpcServer := grpc.NewServer()
	ln, err := net.Listen("tcp", n.ListenAddr)
	if err != nil {
		return errors.New("node start: " + err.Error())
	}

	proto.RegisterNodeServer(grpcServer, n)
	n.log("Starting node")

	// bootstrap the network with a list of already known nodes
	if len(bootstrapNodes) > 0 {
		go n.bootstrapNetwork(bootstrapNodes)
	}

	if n.PrivateKey != nil {
		go n.validatorLoop()
	}

	return grpcServer.Serve(ln)
}

func (n *Node) HandleTransaction(ctx context.Context, tx *proto.Transaction) (*emptypb.Empty, error) {
	peer, _ := peer.FromContext(ctx)
	hash := hex.EncodeToString(types.HashTransaction(tx))

	if n.mempool.Add(tx) {
		n.log("received tx", "from", peer.Addr, "hash", hash, "we", n.ListenAddr)
		go func() {
			if err := n.broadcast(tx); err != nil {
				n.log("broadcast failed", "err", err)
			}
		}()
	}

	return &emptypb.Empty{}, nil
}

func (n *Node) Handshake(ctx context.Context, v *proto.Version) (*proto.Version, error) {
	c, err := makeNodeClient(v.ListenAddr)
	if err != nil {
		return nil, errors.New("handshake: " + err.Error())
	}

	n.addPeer(c, v)

	n.log("received handshake", "version", v.Version, "addr", v.ListenAddr, "height", v.Height)

	return n.getVersion(), nil
}

func (n *Node) validatorLoop() {
	n.log("starting validator loop", "pubKey", n.PrivateKey.Public(), "blocktime", blockTime)
	ticker := time.NewTicker(blockTime)
	for {
		<-ticker.C

		txx := n.mempool.Clear()
		n.log("time to create a new block", "lenTx", len(txx))
	}
}

func (n *Node) broadcast(msg any) error {
	for peer := range n.peers {
		switch v := msg.(type) {
		case *proto.Transaction:
			_, err := peer.HandleTransaction(context.Background(), v)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (n *Node) getVersion() *proto.Version {
	return &proto.Version{
		Version:    n.Version,
		Height:     n.height,
		ListenAddr: n.ListenAddr,
		PeerList:   n.getPeerList(),
	}
}

func (n *Node) getPeerList() []string {
	n.peerLock.RLock()
	defer n.peerLock.RUnlock()

	peerList := []string{}
	for _, v := range n.peers {
		peerList = append(peerList, v.ListenAddr)
	}
	return peerList
}

func (n *Node) dialRemoteNode(addr string) (proto.NodeClient, *proto.Version, error) {
	c, err := makeNodeClient(addr)
	if err != nil {
		return nil, nil, errors.New("bootstrap network: " + err.Error())
	}

	v, err := c.Handshake(context.TODO(), n.getVersion())
	if err != nil {
		return nil, nil, errors.New("handshake: " + err.Error())
	}

	return c, v, nil
}

func (n *Node) canConnectWith(addr string) bool {
	if addr == n.ListenAddr {
		return false
	}

	connectedPeers := n.getPeerList()
	return !slices.Contains(connectedPeers, addr)
}

func (n *Node) bootstrapNetwork(addrs []string) error {
	for _, addr := range addrs {
		n.log("dialing remote node", "remote", addr)
		if !n.canConnectWith(addr) {
			continue
		}
		c, v, err := n.dialRemoteNode(addr)

		if err != nil {
			n.log("failed to connect to bootstrap node", "address", addr, "error", err)
			continue
		}

		n.addPeer(c, v)
	}

	return nil
}

func (n *Node) log(msg string, args ...any) {
	n.logger.Info(
		fmt.Sprintf("[%s] %s", n.ListenAddr, msg),
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
