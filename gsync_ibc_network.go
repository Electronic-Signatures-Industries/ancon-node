package main

import (
	"context"
	"fmt"

	"net"
	"net/http"
	"time"

	"github.com/ipfs/go-graphsync/network"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/light/proxy"
	"github.com/tendermint/tendermint/light/rpc"
	"github.com/tendermint/tendermint/rpc/jsonrpc/server"
)

var sendMessageTimeout = time.Minute * 10

type Receiver interface {
	ReceiveMessage(
		ctx context.Context,
		sender peer.ID,
		incoming gsmsg.GraphSyncMessage)

	ReceiveError(p peer.ID, err error)

	Connected(p peer.ID)
	Disconnected(p peer.ID)
}

type RPCReceiver struct{}

func (r *RPCReceiver) ReceiveMessage(
	ctx context.Context,
	sender peer.ID,
	incoming gsmsg.GraphSyncMessage) {

}

func (r *RPCReceiver) ReceiveError(p peer.ID, err error) {}

func (r *RPCReceiver) Connected(p peer.ID) {}

func (r *RPCReceiver) Disconnected(p peer.ID) {}

// NewFromTM returns a GraphSyncNetwork supported by underlying Libp2p host.
func NewFromTM(
	host proxy.Proxy,
	r network.Receiver,
	logger log.Logger,
	client *rpc.Client,
	address string) *TmGraphSyncNetwork {

	return &TmGraphSyncNetwork{host, r, logger, client, address}
}

// tmGraphSyncNetwork transforms the libp2p host interface, which sends and receives
// NetMessage objects, into the network.GraphSyncNetwork interface.
type TmGraphSyncNetwork struct {
	host proxy.Proxy
	// inbound messages from the network are forwarded to the receiver
	receiver network.Receiver
	log.Logger
	Client *rpc.Client
	Addr   string
	Conn   server.WebsocketManager
}

func (c *TmGraphSyncNetwork) SubscribeProxy(ctx *rpctypes.Context, query string) (*ctypes.ResultSubscribe, error) {
	out, err := c.next.Subscribe(context.Background(), ctx.RemoteAddr(), query)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			select {
			case resultEvent := <-out:
				// We should have a switch here that performs a validation
				// depending on the event's type.
				ctx.WSConn.TryWriteRPCResponse(
					rpctypes.NewRPCSuccessResponse(
						rpctypes.JSONRPCStringID(fmt.Sprintf("%v#event", ctx.JSONReq.ID)),
						resultEvent,
					))
			case <-c.Quit():
				return
			}
		}
	}()

	return &ctypes.ResultSubscribe{}, nil
}

func (p *TmGraphSyncNetwork) listen() (net.Listener, *http.ServeMux, error) {
	mux := http.NewServeMux()
	c := server.DefaultConfig()

	rpcReceiver := RPCReceiver{}
	p.SetDelegate(rpcReceiver)

	//1) Register regular routes.
	//r := RPCRoutes(p.Client)
	r := map[string]*server.RPCFunc{
		"subscribe":       server.NewWSRPCFunc(p.Client.S, "query"),
		"unsubscribe":     server.NewWSRPCFunc(c.UnsubscribeWS, "query"),
		"unsubscribe_all": server.NewWSRPCFunc(c.UnsubscribeAllWS, ""),
	}
	server.RegisterRPCFuncs(mux, r, p.Logger)

	//2) Allow websocket connections.
	wmLogger := p.Logger.With("protocol", "websocket")
	wm := server.NewWebsocketManager(r,
		server.OnDisconnect(func(remoteAddr string) {
			err := p.Client.UnsubscribeAll(context.Background(), remoteAddr)
			if err != nil && err != tmpubsub.ErrSubscriptionNotFound {
				wmLogger.Error("Failed to unsubscribe addr from events", "addr", remoteAddr, "err", err)
			}
		}),
		server.ReadLimit(c.MaxBodyBytes),
	)
	wm.SetLogger(wmLogger)
	mux.HandleFunc("/graphsync", wm.WebsocketHandler)

	if !p.Client.IsRunning() {
		if err := p.Client.Start(); err != nil {
			return nil, mux, fmt.Errorf("can't start client: %w", err)
		}
	}

	// 4) Start listening for new connections.
	listener, err := server.Listen(p.Addr, c)
	if err != nil {
		return nil, mux, err
	}

	return listener, mux, nil
}

func (p *TmGraphSyncNetwork) ListenAndServe() error {
	listener, mux, err := p.listen()
	if err != nil {
		return err
	}
	//p.Listener = listener

	c := server.DefaultConfig()

	return server.Serve(
		listener,
		mux,
		p.Logger,
		c,
	)
}

func (p *TmGraphSyncNetwork) SetDelegate(r Receiver) {

}
