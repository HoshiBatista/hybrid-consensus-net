package network

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"go-hybrid-blockchain/internal/blockchain"
	"go-hybrid-blockchain/internal/consensus"
)

// testNode starts a node on a random localhost port and registers a cleanup
// function that closes it when the test ends.
func testNode(t *testing.T, ctx context.Context) *Node {
	t.Helper()
	n, err := NewNode(ctx, "/ip4/127.0.0.1/tcp/0")
	if err != nil {
		t.Fatalf("NewNode: %v", err)
	}
	t.Cleanup(func() { _ = n.Close() })
	return n
}

// minedBlock returns a PoW-mined block suitable for broadcast tests.
func minedBlock(t *testing.T, prevHash []byte, height int) *blockchain.Block {
	t.Helper()
	b := blockchain.NewBlock([]*blockchain.Transaction{}, prevHash, height)
	if err := consensus.NewProofOfWork(1).MineBlock(b); err != nil {
		t.Fatalf("MineBlock: %v", err)
	}
	return b
}

// connectPeers creates a direct TCP connection from a to b and waits briefly
// for GossipSub to exchange subscription state over the new link.
func connectPeers(t *testing.T, ctx context.Context, a, b *Node) {
	t.Helper()
	if err := a.Host.Connect(ctx, b.AddrInfo()); err != nil {
		t.Fatalf("Host.Connect: %v", err)
	}
	// Allow GossipSub to exchange subscription RPCs. With WithFloodPublish the
	// router sends to all subscribed connected peers, so once the subscription
	// tables are synced any publish is guaranteed to arrive.
	time.Sleep(200 * time.Millisecond)
}

// --- Node lifecycle -------------------------------------------------------

func TestNewNode_HasPeerID(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	n := testNode(t, ctx)
	if n.PeerID() == "" {
		t.Fatal("PeerID must not be empty after NewNode")
	}
}

func TestNewNode_HasListenAddress(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	n := testNode(t, ctx)
	if len(n.Host.Addrs()) == 0 {
		t.Fatal("node must expose at least one listen address")
	}
}

func TestNewNode_AddrInfoConsistent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	n := testNode(t, ctx)
	ai := n.AddrInfo()
	if ai.ID != n.Host.ID() {
		t.Fatalf("AddrInfo.ID %s != Host.ID %s", ai.ID, n.Host.ID())
	}
	if len(ai.Addrs) == 0 {
		t.Fatal("AddrInfo must include at least one address")
	}
}

func TestNode_Close(t *testing.T) {
	ctx := context.Background()
	n, err := NewNode(ctx, "/ip4/127.0.0.1/tcp/0")
	if err != nil {
		t.Fatalf("NewNode: %v", err)
	}
	if err := n.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func TestTwoNodes_HaveDifferentPeerIDs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	a := testNode(t, ctx)
	b := testNode(t, ctx)
	if a.PeerID() == b.PeerID() {
		t.Fatal("two independently created nodes must have different peer IDs")
	}
}

// --- Peer connectivity ---------------------------------------------------

func TestConnect_PeerAppearsInConnectedList(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	a := testNode(t, ctx)
	b := testNode(t, ctx)
	connectPeers(t, ctx, a, b)

	peers := a.ConnectedPeers()
	for _, id := range peers {
		if id == b.Host.ID() {
			return // found — pass
		}
	}
	t.Fatalf("b (%s) not in a's connected peer list: %v", b.Host.ID().ShortString(), peers)
}

// --- BroadcastBlock ------------------------------------------------------

func TestBroadcastBlock_NoError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	n := testNode(t, ctx)
	block := minedBlock(t, []byte("genesis"), 1)
	if err := n.BroadcastBlock(ctx, block); err != nil {
		t.Fatalf("BroadcastBlock: %v", err)
	}
}

func TestBroadcastBlock_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	n := testNode(t, ctx)
	block := minedBlock(t, []byte("genesis"), 1)

	cancelled, cancelNow := context.WithCancel(ctx)
	cancelNow() // cancel before publish
	// Publish on a cancelled context — may or may not error depending on timing;
	// the key assertion is that the call returns (no hang).
	_ = n.BroadcastBlock(cancelled, block)
}

// --- BlockStream + end-to-end gossip -------------------------------------

func TestBlockStream_PeerReceivesBlock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sender := testNode(t, ctx)
	receiver := testNode(t, ctx)
	connectPeers(t, ctx, sender, receiver)

	stream := receiver.BlockStream(ctx)

	block := minedBlock(t, []byte("prev"), 1)
	if err := sender.BroadcastBlock(ctx, block); err != nil {
		t.Fatalf("BroadcastBlock: %v", err)
	}

	select {
	case got := <-stream:
		if got.Height != block.Height {
			t.Fatalf("received Height=%d, want %d", got.Height, block.Height)
		}
	case <-ctx.Done():
		t.Fatal("timeout: receiver never got the block")
	}
}

func TestBlockStream_HashSurvivesRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sender := testNode(t, ctx)
	receiver := testNode(t, ctx)
	connectPeers(t, ctx, sender, receiver)

	stream := receiver.BlockStream(ctx)
	block := minedBlock(t, []byte("prev"), 2)

	if err := sender.BroadcastBlock(ctx, block); err != nil {
		t.Fatalf("BroadcastBlock: %v", err)
	}

	select {
	case got := <-stream:
		if len(got.Hash) == 0 {
			t.Fatal("received block has empty Hash — JSON round-trip broken")
		}
		if len(got.PrevBlockHash) == 0 {
			t.Fatal("received block has empty PrevBlockHash — JSON round-trip broken")
		}
		// Byte-level equality with the original.
		for i, b := range block.Hash {
			if got.Hash[i] != b {
				t.Fatalf("Hash mismatch at byte %d: got %x want %x", i, got.Hash[i], b)
			}
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for block")
	}
}

func TestBlockStream_SelfPublishFiltered(t *testing.T) {
	// A block broadcast by the same node must not appear on its own BlockStream.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	n := testNode(t, ctx)
	stream := n.BlockStream(ctx)
	block := minedBlock(t, []byte("prev"), 1)
	_ = n.BroadcastBlock(ctx, block)

	select {
	case b := <-stream:
		t.Fatalf("self-published block leaked into BlockStream: height=%d", b.Height)
	case <-time.After(500 * time.Millisecond):
		// Expected: nothing arrives.
	}
}

func TestBlockStream_ClosedOnCtxCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	n := testNode(t, ctx)
	stream := n.BlockStream(ctx)
	cancel() // cancel the parent context

	select {
	case _, open := <-stream:
		if open {
			t.Fatal("channel should be closed after context cancellation")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("channel was not closed after context cancellation")
	}
}

func TestBlockStream_MultipleBlocks(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	sender := testNode(t, ctx)
	receiver := testNode(t, ctx)
	connectPeers(t, ctx, sender, receiver)

	stream := receiver.BlockStream(ctx)

	const count = 3
	sent := make([]*blockchain.Block, count)
	for i := range sent {
		sent[i] = minedBlock(t, []byte{byte(i)}, i+1)
		if err := sender.BroadcastBlock(ctx, sent[i]); err != nil {
			t.Fatalf("BroadcastBlock[%d]: %v", i, err)
		}
	}

	received := make(map[int]bool)
	for len(received) < count {
		select {
		case b := <-stream:
			received[b.Height] = true
		case <-ctx.Done():
			t.Fatalf("timeout after receiving %d/%d blocks", len(received), count)
		}
	}
}

// --- JSON round-trip (unit, no network) ----------------------------------

func TestBlock_JSONRoundTrip(t *testing.T) {
	// Verifies that MarshalJSON + UnmarshalJSON preserve all fields faithfully,
	// including the hex-encoded Hash and PrevBlockHash byte slices.
	block := minedBlock(t, []byte("prevhash-bytes"), 7)

	data, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got blockchain.Block
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.Height != block.Height {
		t.Errorf("Height: got %d want %d", got.Height, block.Height)
	}
	if got.Nonce != block.Nonce {
		t.Errorf("Nonce: got %d want %d", got.Nonce, block.Nonce)
	}
	for i, b := range block.Hash {
		if got.Hash[i] != b {
			t.Fatalf("Hash[%d]: got %x want %x", i, got.Hash[i], b)
		}
	}
	for i, b := range block.PrevBlockHash {
		if got.PrevBlockHash[i] != b {
			t.Fatalf("PrevBlockHash[%d]: got %x want %x", i, got.PrevBlockHash[i], b)
		}
	}
}
