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

	peerLock    sync.RWMutex
	peers       map[proto.NodeClient]*proto.Version
	mempool     *Mempool
	chain       *Chain
	validators  *ValidatorSet
	proposeLock sync.Mutex
	seenBlocks  map[string]bool
	seenLock    sync.RWMutex

	proto.UnimplementedNodeServer
}

func NewNode(cfg ServerConfig, chain *Chain, validators *ValidatorSet) *Node {
	return &Node{
		logger:       slog.Default(),
		peers:        make(map[proto.NodeClient]*proto.Version),
		mempool:      NewMempool(),
		chain:        chain,
		validators:   validators,
		seenBlocks:   make(map[string]bool),
		ServerConfig: cfg,
	}
}

func (n *Node) addPeer(c proto.NodeClient, v *proto.Version) {
	n.peerLock.Lock()
	defer n.peerLock.Unlock()

	n.peers[c] = v

	if len(v.PeerList) > 0 {
		go n.bootstrapNetwork(v.PeerList)
	}

	n.log("new peer successfully connected", "height", v.Height, "addr", v.ListenAddr)
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

	if len(bootstrapNodes) > 0 {
		go n.bootstrapNetwork(bootstrapNodes)
	}

	if n.PrivateKey != nil {
		// Wait for network to settle, then check if we should propose
		go func() {
			time.Sleep(2 * time.Second)
			n.maybeProposeBlock()
		}()
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

// maybeProposeBlock checks if it's our turn to propose the next block.
// If yes, creates the block, adds it to our chain, and broadcasts it.
func (n *Node) maybeProposeBlock() {
	n.proposeLock.Lock()
	defer n.proposeLock.Unlock()

	height := int32(n.chain.Height() + 1)
	proposer := n.validators.GetProposer(height)

	if proposer.String() != n.PrivateKey.Public().String() {
		return
	}

	txx := n.mempool.Clear()
	n.log("time to create a new block", "height", height, "lenTx", len(txx))

	block, err := n.createBlock(height, txx)
	if err != nil {
		n.log("failed to create block", "err", err)
		return
	}

	if err := n.chain.AddBlock(block); err != nil {
		n.log("failed to add block to local chain", "err", err)
		return
	}

	blockHash := hex.EncodeToString(types.HashBlock(block))
	n.seenLock.Lock()
	n.seenBlocks[blockHash] = true
	n.seenLock.Unlock()

	n.log("block created", "height", height, "hash", blockHash)

	go func() {
		if err := n.broadcast(block); err != nil {
			n.log("broadcast block failed", "err", err)
		}
	}()
}

func (n *Node) createBlock(height int32, txx []*proto.Transaction) (*proto.Block, error) {
	currentBlock, err := n.chain.GetBlockByHeight(n.chain.Height())
	if err != nil {
		return nil, fmt.Errorf("failed to get current block: %w", err)
	}

	prevHash := types.HashBlock(currentBlock)

	header := &proto.Header{
		Version:   1,
		Height:    height,
		PrevHash:  prevHash,
		Timestamp: time.Now().UnixNano(),
	}

	block := &proto.Block{
		Header:       header,
		Transactions: txx,
	}

	types.SignBlock(*n.PrivateKey, block)

	return block, nil
}

func (n *Node) HandleBlock(ctx context.Context, block *proto.Block) (*emptypb.Empty, error) {
	peer, _ := peer.FromContext(ctx)
	hash := hex.EncodeToString(types.HashBlock(block))

	n.seenLock.Lock()
	if n.seenBlocks[hash] {
		n.seenLock.Unlock()
		return &emptypb.Empty{}, nil
	}
	n.seenBlocks[hash] = true
	n.seenLock.Unlock()

	if err := n.chain.AddBlock(block); err != nil {
		n.log("block validation failed", "from", peer.Addr, "hash", hash, "height", block.Header.Height, "err", err)
		return nil, err
	}

	n.log("block accepted", "from", peer.Addr, "hash", hash, "height", block.Header.Height, "chainHeight", n.chain.Height())

	// After accepting a block, check if we should propose the next one
	go n.maybeProposeBlock()

	// Re-broadcast to peers who haven't seen it
	go func() {
		if err := n.broadcast(block); err != nil {
			n.log("rebroadcast block failed", "err", err)
		}
	}()

	return &emptypb.Empty{}, nil
}

func (n *Node) broadcast(msg any) error {
	for peer := range n.peers {
		switch v := msg.(type) {
		case *proto.Transaction:
			_, err := peer.HandleTransaction(context.Background(), v)
			if err != nil {
				return err
			}
		case *proto.Block:
			_, err := peer.HandleBlock(context.Background(), v)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (n *Node) getVersion() *proto.Version {
	v := &proto.Version{
		Version:    n.Version,
		Height:     int32(n.chain.Height()),
		ListenAddr: n.ListenAddr,
		PeerList:   n.getPeerList(),
	}

	for _, pk := range n.validators.validators {
		v.Validators = append(v.Validators, pk.Bytes())
	}

	return v
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
