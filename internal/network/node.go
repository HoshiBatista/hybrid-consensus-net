// Package network implements the P2P layer for the hybrid blockchain node.
// It uses libp2p as the transport and GossipSub as the publish-subscribe
// protocol for broadcasting mined blocks to connected peers.
//
// # Architecture
//
//	Node
//	  ├── libp2p Host   — manages TCP/QUIC connections, TLS handshake, peer routing
//	  ├── GossipSub     — epidemic broadcast; blocks are published to BlockTopic
//	  └── mDNS service  — zero-configuration peer discovery on the local network
//
// # Peer discovery
//
// mDNS (RFC 6762) is used for local-network peer discovery. When a peer is
// found, the node connects to it immediately. No bootstrap nodes or DHT are
// required for a LAN deployment. For WAN deployments a separate bootstrap
// mechanism (e.g. libp2p-kad-dht) should be added.
//
// # Block gossip
//
// GossipSub is configured with WithFloodPublish(true) so that a block
// published by any node is immediately forwarded to every directly-connected
// peer that has subscribed to the topic, regardless of the mesh state. This
// guarantees reliable delivery in small networks (< 6 peers) where GossipSub's
// default mesh would never form.
package network

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	libp2p "github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"

	"go-hybrid-blockchain/internal/blockchain"
)

const (
	// BlockTopic is the GossipSub topic name for newly mined blocks.
	// The version suffix lets future protocol changes break cleanly.
	BlockTopic = "hybrid-blockchain/blocks/v1"

	// mdnsServiceTag is the mDNS service name used for local peer discovery.
	// Nodes on the same LAN with the same tag will find each other automatically.
	mdnsServiceTag = "hybrid-blockchain-mdns"
)

// Node is the P2P node. It owns a libp2p Host, a GossipSub router, a block
// topic subscription, and an mDNS discovery service.
type Node struct {
	Host     host.Host
	ps       *pubsub.PubSub
	topic    *pubsub.Topic
	sub      *pubsub.Subscription
	mdnsSvc  mdns.Service
}

// discoveryNotifee handles mDNS peer-found events. Each discovered peer is
// connected to immediately; errors are logged but do not stop discovery.
type discoveryNotifee struct{ h host.Host }

// HandlePeerFound satisfies mdns.Notifee. It is called from a background
// goroutine managed by the mDNS service.
func (d *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	if pi.ID == d.h.ID() {
		return // ignore self-announcements
	}
	log.Printf("[mdns] found peer %s — connecting", pi.ID.ShortString())
	if err := d.h.Connect(context.Background(), pi); err != nil {
		log.Printf("[mdns] connect to %s: %v", pi.ID.ShortString(), err)
	}
}

// NewNode creates a fully-initialised P2P node that:
//  1. Listens on listenAddr (e.g. "/ip4/0.0.0.0/tcp/9000").
//  2. Creates a GossipSub router and joins the block broadcast topic.
//  3. Starts an mDNS service for zero-config local peer discovery.
//
// Additional pubsub.Option values can be passed to tune the GossipSub router
// (e.g. to lower mesh parameters for unit tests).
func NewNode(ctx context.Context, listenAddr string, psOpts ...pubsub.Option) (*Node, error) {
	// --- libp2p host ---------------------------------------------------------
	h, err := libp2p.New(libp2p.ListenAddrStrings(listenAddr))
	if err != nil {
		return nil, fmt.Errorf("network: create host: %w", err)
	}

	// --- GossipSub -----------------------------------------------------------
	// WithFloodPublish ensures every directly-connected subscribed peer
	// receives a block even when the gossip mesh has not yet stabilised (common
	// with fewer than 6 peers).
	defaultOpts := []pubsub.Option{pubsub.WithFloodPublish(true)}
	ps, err := pubsub.NewGossipSub(ctx, h, append(defaultOpts, psOpts...)...)
	if err != nil {
		_ = h.Close()
		return nil, fmt.Errorf("network: create gossipsub: %w", err)
	}

	topic, err := ps.Join(BlockTopic)
	if err != nil {
		_ = h.Close()
		return nil, fmt.Errorf("network: join topic %q: %w", BlockTopic, err)
	}

	sub, err := topic.Subscribe()
	if err != nil {
		_ = topic.Close()
		_ = h.Close()
		return nil, fmt.Errorf("network: subscribe to %q: %w", BlockTopic, err)
	}

	// --- mDNS discovery ------------------------------------------------------
	svc := mdns.NewMdnsService(h, mdnsServiceTag, &discoveryNotifee{h: h})
	if err := svc.Start(); err != nil {
		sub.Cancel()
		_ = topic.Close()
		_ = h.Close()
		return nil, fmt.Errorf("network: start mdns: %w", err)
	}

	log.Printf("[p2p] node ready\n  peer id : %s\n  listen  : %v", h.ID(), h.Addrs())

	return &Node{
		Host:    h,
		ps:      ps,
		topic:   topic,
		sub:     sub,
		mdnsSvc: svc,
	}, nil
}

// BroadcastBlock JSON-encodes block and publishes it to the block gossip topic.
// All peers subscribed to BlockTopic will receive the message.
// The block's Hash and PrevBlockHash are preserved faithfully through the
// hex-aware MarshalJSON / UnmarshalJSON pair on blockchain.Block.
func (n *Node) BroadcastBlock(ctx context.Context, block *blockchain.Block) error {
	data, err := json.Marshal(block)
	if err != nil {
		return fmt.Errorf("network: marshal block height=%d: %w", block.Height, err)
	}
	if err := n.topic.Publish(ctx, data); err != nil {
		return fmt.Errorf("network: publish block height=%d: %w", block.Height, err)
	}
	return nil
}

// BlockStream returns a channel that emits blocks received from remote peers.
// Messages published by this node are filtered out (self-loop prevention).
// The channel is closed when ctx is cancelled or Close is called.
func (n *Node) BlockStream(ctx context.Context) <-chan *blockchain.Block {
	ch := make(chan *blockchain.Block, 16)
	go func() {
		defer close(ch)
		for {
			msg, err := n.sub.Next(ctx)
			if err != nil {
				return // ctx cancelled or subscription closed
			}
			if msg.ReceivedFrom == n.Host.ID() {
				continue // skip own messages re-delivered by the router
			}
			var block blockchain.Block
			if err := json.Unmarshal(msg.Data, &block); err != nil {
				log.Printf("[p2p] drop malformed block message: %v", err)
				continue
			}
			select {
			case ch <- &block:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch
}

// AddrInfo returns the peer.AddrInfo for this node, suitable for passing to
// another host's Connect call (used in tests and for manual peering).
func (n *Node) AddrInfo() peer.AddrInfo {
	return peer.AddrInfo{ID: n.Host.ID(), Addrs: n.Host.Addrs()}
}

// PeerID returns the human-readable peer ID string.
func (n *Node) PeerID() string { return n.Host.ID().String() }

// ConnectedPeers returns the IDs of all currently connected peers.
func (n *Node) ConnectedPeers() []peer.ID { return n.Host.Network().Peers() }

// Close cancels the block subscription, closes the topic, stops mDNS, and
// shuts down the libp2p host. Errors from each step are joined and returned.
func (n *Node) Close() error {
	n.sub.Cancel()
	var errs []error
	if err := n.topic.Close(); err != nil {
		errs = append(errs, fmt.Errorf("topic: %w", err))
	}
	if err := n.mdnsSvc.Close(); err != nil {
		errs = append(errs, fmt.Errorf("mdns: %w", err))
	}
	if err := n.Host.Close(); err != nil {
		errs = append(errs, fmt.Errorf("host: %w", err))
	}
	if len(errs) > 0 {
		return fmt.Errorf("network Close: %v", errs)
	}
	return nil
}
