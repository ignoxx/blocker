# blocker

custom UTXO blockchain implementation from scratch

## features
- [x] ED25519 crypto for keys and signatures
- [x] UTXO transaction model
- [x] Protobuf encoding
- [x] gRPC transport (p2p gossip between nodes)
- [x] Proof of Stake (PoS) consensus — round-robin, event-driven
- [x] Wallet CLI — create wallet, query balance/transactions, send coins
- [x] Block validation — height, prevHash, proposer, signature, UTXO double-spend

## run the network

```bash
# Start 3 validator nodes
make run
```

Nodes will bootstrap, connect to each other, and produce blocks only when there are transactions to include.

## cli

```bash
# Create a new wallet (prints 64-char seed + address)
make cli create

# Query balance for an address
make cli balance <hex-address>
make cli -node :4000 balance 36f8343eaa41afbff509addde0f5cb4a0691295b

# Query transactions for an address
make cli txs <hex-address>

# Send coins (uses your private key seed + recipient address + amount)
make cli send <your-priv-key-seed> <recipient-address> <amount>
make cli send 0f71b4d... 72974c1... 100
```

## genesis

The genesis block allocates **1000** coins to a fixed address derived from the hardcoded seed. This is the address you can spend from in the demo:

```
Address: 36f8343eaa41afbff509addde0f5cb4a0691295b
```

## not production-ready (high-level gaps)

this is a learning project. a production blockchain would need at least the following:

### consensus
- **weighted proposer selection** — validators chosen by stake amount, not equal round-robin
- **fork choice rule** — pick the heaviest/best chain when forks occur
- **block finality** — blocks are not final; reorgs can happen at any time
- **slashing** — no economic penalties for double-signing or invalid blocks
- **validator rotation** — validator set is hardcoded; no dynamic join/leave

### networking
- **block sync for late joiners** — new nodes starting behind the tip never catch up
- **out-of-order block buffering** — blocks received too far ahead are dropped, not buffered
- **peer scoring / banning** — no mechanism to disconnect or penalize bad peers
- **connection encryption** — gRPC runs without TLS

### data layer
- **persistent storage** — everything is in-memory; restart loses all data
- **state snapshots / pruning** — chain grows forever, no old state cleanup
- **merkle proofs** — can't prove inclusion of a tx to a light client

### transactions / vm
- **no smart contracts / VM** — only simple value transfers
- **fixed fees** — no fee market or gas pricing
- **no nonce / replay protection** — same tx can be replayed unless UTXO is spent

### tooling
- **no block explorer** — no way to inspect chain state visually
- **no metrics / monitoring** — no prometheus, health checks, or alerts