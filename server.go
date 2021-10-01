package main

import (
	"context"
	"fmt"

	"os"
	"time"

	"github.com/tendermint/tendermint/libs/log"
	httpprovider "github.com/tendermint/tendermint/light/provider/http"
	dbm "github.com/tendermint/tm-db"
	badger "github.com/tendermint/tm-db/badgerdb"

	blockstore "github.com/ipfs/go-ipfs-blockstore"
	"github.com/tendermint/tendermint/light/provider"
	dbs "github.com/tendermint/tendermint/light/store/db"

	"github.com/tendermint/tendermint/light"
	"github.com/tendermint/tendermint/light/proxy"
	"github.com/tendermint/tendermint/rpc/jsonrpc/server"

	graphsync "github.com/ipfs/go-graphsync/impl"
	gsnet "github.com/ipfs/go-graphsync/network"
	"github.com/ipfs/go-graphsync/storeutil"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/routing"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	noise "github.com/libp2p/go-libp2p-noise"
)

// example of libp2p host - https://github.com/libp2p/go-libp2p/tree/master/examples/libp2p-host
func main() {

	run()
	/*
		// TODO: json-rpc https://github.com/mrFokin/jrpc/blob/master/jrpc_test.go
		e := echo.New()
		e.GET("/", func(c echo.Context) error {
			return c.String(http.StatusOK, "Hello, World!")
		})
		e.Logger.Fatal(e.Start(":1323"))*/
}

func run() {
	// The context governs the lifetime of the libp2p node.
	// Cancelling it will stop the the host.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set your own keypair
	priv, _, err := crypto.GenerateKeyPair(
		crypto.Ed25519, // Select your key type. Ed25519 are nice short
		-1,             // Select key length when possible (i.e. RSA).
	)
	if err != nil {
		panic(err)
	}

	var dht *kaddht.IpfsDHT
	newDHT := func(h host.Host) (routing.PeerRouting, error) {
		var err error
		dht, err = kaddht.New(ctx, h)
		return dht, err
	}

	gsynchost, err := libp2p.New(
		ctx,
		// Use the keypair we generated
		libp2p.Identity(priv),
		libp2p.Security(noise.ID, noise.New),
		// Multiple listen addresses
		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/9000",      // regular tcp connections
			"/ip4/0.0.0.0/udp/9000/quic", // a UDP endpoint for the QUIC transport
		),
		// support TLS connections
		// Let's prevent our peer from having too many
		// connections by attaching a connection manager.
		libp2p.ConnectionManager(connmgr.NewConnManager(
			100,         // Lowwater
			400,         // HighWater,
			time.Minute, // GracePeriod
		)),
		// Attempt to open ports using uPNP for NATed hosts.
		libp2p.NATPortMap(),
		// Let this host use the DHT to find other hosts
		libp2p.Routing(newDHT),
		// Let this host use relays and advertise itself on relays if
		// it finds it is behind NAT. Use libp2p.Relay(options...) to
		// enable active relays and more.
		libp2p.EnableAutoRelay(),
		// If you want to help other peers to figure out if they are behind
		// NATs, you can launch the server-side of AutoNAT too (AutoRelay
		// already runs the client)
		//
		// This service is highly rate-limited and should not cause any
		// performance issues.
		libp2p.EnableNATService(),
	)
	if err != nil {
		panic(err)
	}
	defer gsynchost.Close()

	// The last step to get fully up and running would be to connect to
	// bootstrap peers (or any other peers). We leave this commented as
	// this is an example and the peer will die as soon as it finishes, so
	// it is unnecessary to put strain on the network.

	/*
		// This connects to public bootstrappers
		for _, addr := range dht.DefaultBootstrapPeers {
			pi, _ := peer.AddrInfoFromP2pAddr(addr)
			// We ignore errors as some bootstrap peers may be down
			// and that is fine.
			gsynchost.Connect(ctx, *pi)
		}
	*/
	fmt.Printf("Hello World, my hosts ID is %s\n", gsynchost.ID())

	var bs blockstore.Blockstore

	network := gsnet.NewFromLibp2pHost(gsynchost)
	lsys := storeutil.LinkSystemForBlockstore(bs)
	// add carv1
	_ = graphsync.New(ctx, network, lsys)

	db, err := badger.NewDB("anconnode", "/tmp/badger")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open badger db: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	h := 4
	hash := []byte("FFABCB1821479586177683DA92044DEA9E51232DAEC3AAA54D244E5B00F28B43")

	node, err := newLightTendermint(ctx, "placehere", "http://localhost:26657", "http://localhost:26657", h, hash, dbm.DB(db))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(2)
	}

	// rpc
	c := server.DefaultConfig()
	proxy, err := proxy.NewProxy(node, "http://localhost:8899", "http://localhost:26657", c, log.NewNopLogger())
	proxy.ListenAndServe()

}

func newLightTendermint(ctx context.Context, chainID string,
	primary string, witness string, height int, hash []byte, db dbm.DB) (*light.Client, error) {

	primaryNode, _ := httpprovider.New(chainID, primary)
	witnessNode, _ := httpprovider.New(chainID, witness)
	c, _ := light.NewClient(
		ctx,
		chainID,
		light.TrustOptions{
			Period: 504 * time.Hour, // 21 days
			Height: int64(height),
			Hash:   hash,
		},
		primaryNode,
		[]provider.Provider{witnessNode},
		dbs.New(db, "ancon-node"),
		light.Logger(log.TestingLogger()),
	)
	//_, err := c.Update(ctx, time.Now())

	return c, nil
}
