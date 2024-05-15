package main

import (
	"context"
	"encoding/gob"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/multiformats/go-multiaddr"
)

const (
	protocolID         = "p2pconnect"
	discoveryNamespace = "p2pconnect"
)

type SID struct {
	ID string
}

type discoveryNotifee struct {
	h host.Host
}

func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	fmt.Printf("discovered new peer %s\n", pi.ID)
	err := n.h.Connect(context.Background(), pi)
	if err != nil {
		fmt.Printf("error connecting to peer %s: %s\n", pi.ID, err)
	}
}

func WriteConnectedNodes(s network.Stream) {
	strID := SID{
		s.ID(),
	}
	enc := gob.NewEncoder(s)
	if err := enc.Encode(strID); err != nil {
		fmt.Println(err)
	}
}

func ReadConnectedNodes(s network.Stream) {
	var strID SID
	dec := gob.NewDecoder(s)
	if err := dec.Decode(&strID); err != nil {
		fmt.Println(err)
	}
	fmt.Println(strID)
}

func main() {
	peerAddr := flag.String("peer-address", "", "connect to a peer")
	flag.Parse()

	ctx := context.Background()
	host, err := libp2p.New(libp2p.ListenAddrStrings(
		fmt.Sprint("/ip4/0.0.0.0/tcp/8080"),
	))
	if err != nil {
		panic(err)
	}
	defer host.Close()

	host.SetStreamHandler(protocolID, func(s network.Stream) {
		go WriteConnectedNodes(s)
		go ReadConnectedNodes(s)
	})

	s := mdns.NewMdnsService(host, discoveryNamespace, &discoveryNotifee{h: host})
	s.Start()

	fmt.Println("Addresses:", host.Addrs())
	fmt.Println("ID:", host.ID())
	fmt.Printf("P2P Address String: %v/p2p/%v", host.Addrs()[1], host.ID())

	if *peerAddr != "" {
		fmt.Println(peerAddr)
		peerMA, err := multiaddr.NewMultiaddr(*peerAddr)
		if err != nil {
			panic(err)
		}
		defer host.Close()

		peerAI, err := peer.AddrInfoFromP2pAddr(peerMA)
		if err != nil {
			panic(err)
		}
		defer host.Close()

		err = host.Connect(ctx, *peerAI)
		if err != nil {
			panic(err)
		}
		defer host.Close()
		fmt.Println("\nConnected Peer :")
		fmt.Println(peerAI.String())

		fmt.Println("Setting up stream...")
		s, err := host.NewStream(ctx, peerAI.ID, protocolID)
		if err != nil {
			panic(err)
		}
		defer host.Close()
		go WriteConnectedNodes(s)
		go ReadConnectedNodes(s)

	}

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGKILL, syscall.SIGINT)
	<-sigCh
}
