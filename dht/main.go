package main

import (
	"context"
	"dht/api"
	"dht/store"
	"fmt"
	"github.com/anacrolix/dht/v2"
	"github.com/anacrolix/dht/v2/bep44"
	"github.com/anacrolix/dht/v2/int160"
	"github.com/anacrolix/dht/v2/krpc"
	peer_store "github.com/anacrolix/dht/v2/peer-store"
	_ "github.com/anacrolix/envpprof"
	"github.com/anacrolix/log"
	"net"
	"net/netip"
	"os"
	"time"
)

func main() {
	conn, err := net.ListenPacket("udp", fmt.Sprintf("0.0.0.0:%s", os.Getenv("DHT_PORT")))
	if err != nil {
		panic(err)
	}
	defer func(conn net.PacketConn) {
		err := conn.Close()
		if err != nil {
			panic(err)
		}
	}(conn)

	cfg := new(dht.ServerConfig)
	cfg.InitNodeId()
	cfg.Conn = conn
	cfg.Exp = time.Duration(10000) * time.Hour
	cfg.DefaultWant = []krpc.Want{krpc.WantNodes, krpc.WantNodes6}
	cfg.Logger = log.Default.FilterLevel(log.Debug)
	cfg.NoSecurity = true
	cfg.StartingNodes = func() ([]dht.Addr, error) {
		return []dht.Addr{dht.NewAddr(net.UDPAddrFromAddrPort(netip.MustParseAddrPort(os.Getenv("STARTING_NODE"))))}, nil
	}
	cfg.WaitToReply = true

	valueStore := store.NewMemory()
	peerStore := &peer_store.InMemory{RootId: int160.FromByteArray(cfg.NodeId)}
	cfg.Store = valueStore
	cfg.PeerStore = peerStore

	dhtSrv, err := dht.NewServer(cfg)
	if err != nil {
		panic(err)
	}
	log.Printf("dht server on %s, ID is %x", dhtSrv.Addr(), dhtSrv.ID())
	defer dhtSrv.Close()

	go func() {
		if tried, err := dhtSrv.Bootstrap(); err != nil {
			log.Printf("error bootstrapping: %s", err)
		} else {
			log.Printf("finished bootstrapping: %#v", tried)
		}
	}()

	// announce all keys every 10 seconds
	go func() {
		for range time.Tick(time.Second * 10) {
			for key, _ := range valueStore.GetAll() {
				_, _ = dhtSrv.AnnounceTraversal(key, dht.AnnouncePeer(dht.AnnouncePeerOpts{
					Port: int(netip.MustParseAddrPort(dhtSrv.Addr().String()).Port()),
				}))
			}
		}
	}()

	// download new values every minute
	go func() {
		for key, peers := range peerStore.GetAll() {
			_, err = valueStore.Get(key)
			if err == bep44.ErrItemNotFound && len(peers) > 0 {
				res := dhtSrv.Get(context.TODO(), dht.NewAddr(peers[0].NodeAddr.UDP()), key, nil, dht.QueryRateLimiting{})
				if (res.ToError() != nil) || (res.Reply.R == nil) || (res.Reply.R.V == nil) {
					continue
				}
				bytes, err := res.Reply.R.V.MarshalBencode()
				if err != nil {
					continue
				}
				storeItem := bep44.Item{Key: (*[20]byte)(&key), V: string(bytes)}
				_ = valueStore.Put(&storeItem)
			}
		}
	}()

	apiSrv := api.NewServer(dhtSrv, valueStore, peerStore)
	log.Printf("Start serving on %s\n", apiSrv.Addr)
	err = apiSrv.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
