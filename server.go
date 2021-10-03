package main

import (
	"context"
	"fmt"

	"time"

	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	gsync "github.com/ipfs/go-graphsync"
	graphsync "github.com/ipfs/go-graphsync/impl"
	gsmsg "github.com/ipfs/go-graphsync/message"
	gsnet "github.com/ipfs/go-graphsync/network"
	blockstore "github.com/ipld/go-car/v2/blockstore"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/ipld/go-ipld-prime/traversal/selector/builder"
	"github.com/multiformats/go-multiaddr"
	linkstore "github.com/proofzero/go-ipld-linkstore"

	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	peer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	noise "github.com/libp2p/go-libp2p-noise"
)

// example of libp2p host - https://github.com/libp2p/go-libp2p/tree/master/examples/libp2p-host
func main() {
	// The context governs the lifetime of the libp2p node.
	// Cancelling it will stop the the host.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h1 := newPeer(ctx, "/ip4/0.0.0.0/tcp/7779")
	h2 := newPeer(ctx, "/ip4/0.0.0.0/tcp/7777")

	run(ctx, h1,
		fmt.Sprintf("%s/p2p/%s", h2.Addrs()[0].String(), h2.ID().Pretty()))
	//  	run(ctx, h2, h1.Addrs()[0].String())
	// run(ctx, h2,
	// 	fmt.Sprintf("%s/p2p/%s", h1.Addrs()[0].String(), h1.ID().Pretty()))
}

func newPeer(ctx context.Context, addr string) host.Host {
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
		libp2p.ListenAddrStrings(addr),

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
	//	defer gsynchost.Close()
	return gsynchost
}
func run(ctx context.Context, gsynchost host.Host, bootstrap string) string {

	// The last step to get fully up and running would be to connect to
	// bootstrap peers (or any other peers). We leave this commented as
	// this is an example and the peer will die as soon as it finishes, so
	// it is unnecessary to put strain on the network.

	// This connects to public bootstrappers
	// for _, addr := range dht.DefaultBootstrapPeers {
	pi, _ := peer.AddrInfoFromP2pAddr(multiaddr.StringCast(bootstrap))
	// We ignore errors as some bootstrap peers may be down
	// and that is fine.
	gsynchost.Connect(ctx, *pi)
	// }

	fmt.Printf("Hello World, my hosts ID is %s\n", gsynchost.ID())

	sls := linkstore.NewStorageLinkSystemWithNewStorage(cidlink.DefaultLinkSystem())
	network := gsnet.NewFromLibp2pHost(gsynchost)

	ssb := builder.NewSelectorSpecBuilder(basicnode.Prototype.Any)
	selector := ssb.ExploreAll(ssb.Matcher()).Node()
	//add carv1
	var exchange gsync.GraphExchange
	exchange = graphsync.New(ctx, network, sls.LinkSystem)

	c, _ := cid.Parse("bafyreigiumx5ficjmdwdgpsxddfeyx2vh6cbod5s454pqeaosue33w2fpq")
	link := cidlink.Link{Cid: c}

	finalResponseStatusChan := make(chan gsync.ResponseStatusCode, 1)
	exchange.RegisterCompletedResponseListener(func(p peer.ID, request gsync.RequestData, status gsync.ResponseStatusCode) {
		select {
		case finalResponseStatusChan <- status:
			fmt.Sprintf("%s", status)
		default:
		}
	})

	r := &receiver{
		messageReceived: make(chan receivedMessage),
	}

	network.SetDelegate(r)
	err := network.ConnectTo(ctx, pi.ID)
	if err != nil {
		panic(err)
	}
	pgChan, errChan := exchange.Request(ctx, pi.ID, link, selector)
	VerifyHasErrors(ctx, errChan)
	PrintProgress(ctx, pgChan)

	// var received gsmsg.GraphSyncMessage
	// var receivedBlocks []blocks.Block
	// for {
	// 	var message receivedMessage

	// 	sender := message.sender
	// 	received = message.message
	// 	fmt.Sprintf("%s %s", sender.String(), received)
	// 	receivedBlocks = append(receivedBlocks, received.Blocks()...)
	// 	receivedResponses := received.Responses()
	// 	fmt.Sprintf("%s", receivedResponses[0].Status())
	// 	if receivedResponses[0].Status() != gsync.PartialResponse {
	// 		break
	// 	}
	// }

	// TODO: json-rpc https://github.com/mrFokin/jrpc/blob/master/jrpc_test.go
	return ""
}

//WriteCAR
func ReadCAR() ([]cid.Cid, blocks.Block, datamodel.Node, error) {
	//lsys := linkstore.NewStorageLinkSystemWithNewStorage(cidlink.DefaultLinkSystem())
	ssb := builder.NewSelectorSpecBuilder(basicnode.Prototype.Any)
	selector := ssb.ExploreAll(ssb.Matcher()).Node()

	// car := carv1.NewSelectiveCar(context.Background(),
	// 	lsys.ReadStore,
	// 	[]carv1.Dag{{
	// 		Root:     root,
	// 		Selector: selector,
	// 	}})
	// file, err := os.ReadFile(filename)
	// if err != nil {
	// 	return err
	// }

	robs, _ := blockstore.OpenReadOnly("/home/dallant/Code/ancon-node/dagbridge-block-239-begin.car",
		blockstore.UseWholeCIDs(true),
	)

	roots, err := robs.Roots()

	res, _ := robs.Get(roots[0])

	return roots, res, selector, err
}

type receivedMessage struct {
	message gsmsg.GraphSyncMessage
	sender  peer.ID
}

// Receiver is an interface for receiving messages from the GraphSyncNetwork.
type receiver struct {
	messageReceived chan receivedMessage
}

func (r *receiver) ReceiveMessage(
	ctx context.Context,
	sender peer.ID,
	incoming gsmsg.GraphSyncMessage) {

	select {
	case <-ctx.Done():
	case r.messageReceived <- receivedMessage{incoming, sender}:
	}
}

func (r *receiver) ReceiveError(_ peer.ID, err error) {
	fmt.Println("got receive err")
}

func (r *receiver) Connected(p peer.ID) {
}

func (r *receiver) Disconnected(p peer.ID) {
}

// VerifyHasErrors verifies that at least one error was sent over a channel
func VerifyHasErrors(ctx context.Context, errChan <-chan error) error {
	errCount := 0
	for {
		select {
		case e, ok := <-errChan:
			if ok {
				return nil
			} else {
				return e
			}
			errCount++
		case <-ctx.Done():
		}
	}
}

// VerifyHasErrors verifies that at least one error was sent over a channel
func PrintProgress(ctx context.Context, pgChan <-chan gsync.ResponseProgress) {
	errCount := 0
	for {
		select {
		case data, ok := <-pgChan:
			if ok {
				fmt.Sprintf("path: %s, last path: %s", data.Path.String(), data.LastBlock.Path.String())
			}
			errCount++
		case <-ctx.Done():
		}
	}
}
