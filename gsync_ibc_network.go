package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/apex/log"
	gsmsg "github.com/ipfs/go-graphsync/message"
	"github.com/ipfs/go-graphsync/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-msgio"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/tendermint/tendermint/light/proxy"
	"github.com/tendermint/tendermint/light/rpc"
)

var sendMessageTimeout = time.Minute * 10

// NewFromTM returns a GraphSyncNetwork supported by underlying Libp2p host.
func NewFromTM(host proxy.Proxy) network.GraphSyncNetwork {
	graphSyncNetwork := tmGraphSyncNetwork{
		host: host,
	}

	return &graphSyncNetwork
}

// tmGraphSyncNetwork transforms the libp2p host interface, which sends and receives
// NetMessage objects, into the network.GraphSyncNetwork interface.
type tmGraphSyncNetwork struct {
	host proxy.Proxy
	// inbound messages from the network are forwarded to the receiver
	receiver network.Receiver
}

type streamMessageSender struct {
	s *rpc.Client
}

func (s *streamMessageSender) Close() error {
	return s.s.Stop()

}

func (s *streamMessageSender) Reset() error {
	return s.s.Reset()
}

func (s *streamMessageSender) SendMsg(ctx context.Context, msg gsmsg.GraphSyncMessage) error {
	return msgToStream(ctx, s.s, msg)
}

func msgToStream(ctx context.Context, s *rpc.Client, msg gsmsg.GraphSyncMessage) error {
	log.Debugf("Outgoing message with %d requests, %d responses, and %d blocks",
		len(msg.Requests()), len(msg.Responses()), len(msg.Blocks()))

	deadline := time.Now().Add(sendMessageTimeout)
	if dl, ok := ctx.Deadline(); ok {
		deadline = dl
	}
	if err := s.SetWriteDeadline(deadline); err != nil {
		log.Warnf("error setting deadline: %s", err)
	}

	switch s.Protocol() {
	case network.ProtocolGraphsync:
		if err := msg.ToNet(s); err != nil {
			log.Debugf("error: %s", err)
			return err
		}
	default:
		return fmt.Errorf("unrecognized protocol on remote: %s", s.Protocol())
	}

	if err := s.SetWriteDeadline(time.Time{}); err != nil {
		log.Warnf("error resetting deadline: %s", err)
	}
	return nil
}

func (gsnet *tmGraphSyncNetwork) NewMessageSender(ctx context.Context, p peer.ID) (network.MessageSender, error) {
	s, err := gsnet.newStreamToPeer(ctx, p)
	if err != nil {
		return nil, err
	}

	return &streamMessageSender{s: s}, nil
}

func (gsnet *tmGraphSyncNetwork) newStreamToPeer(ctx context.Context, p peer.ID) (*rpc.Client, error) {
	return gsnet.host.NewStream(ctx, p, ProtocolGraphsync)
}

func (gsnet *tmGraphSyncNetwork) SendMessage(
	ctx context.Context,
	p peer.ID,
	outgoing gsmsg.GraphSyncMessage) error {

	s, err := gsnet.newStreamToPeer(ctx, p)
	if err != nil {
		return err
	}

	if err = msgToStream(ctx, s, outgoing); err != nil {
		_ = s.Reset()
		return err
	}

	return s.Close()
}

func (gsnet *tmGraphSyncNetwork) SetDelegate(r Receiver) {
	gsnet.receiver = r
	gsnet.host.SetStreamHandler(ProtocolGraphsync, gsnet.handleNewStream)
	gsnet.host.Network().Notify((*tmGraphSyncNotifee)(gsnet))
}

func (gsnet *tmGraphSyncNetwork) ConnectTo(ctx context.Context, p peer.ID) error {
	return gsnet.host.Connect(ctx, peer.AddrInfo{ID: p})
}

// handleNewStream receives a new stream from the network.
func (gsnet *tmGraphSyncNetwork) handleNewStream(s *rpc.Client) {
	///	defer s.Close()
	r, _ := s.SubscribeWS(context.Background(), "topic")
	proxy.RPCRoutes(s)
	if gsnet.receiver == nil {
		_ = s.Reset()
		return
	}

	reader := msgio.NewVarintReaderSize(s, network.MessageSizeMax)
	for {
		received, err := gsmsg.FromMsgReader(reader)
		p := s.Conn().RemotePeer()

		if err != nil {
			if err != io.EOF {
				_ = s.Reset()
				go gsnet.receiver.ReceiveError(p, err)
				log.Debugf("graphsync net handleNewStream from %s error: %s", s.Conn().RemotePeer(), err)
			}
			return
		}

		ctx := context.Background()
		log.Debugf("graphsync net handleNewStream from %s", s.Conn().RemotePeer())
		gsnet.receiver.ReceiveMessage(ctx, p, received)
	}
}

func (gsnet *tmGraphSyncNetwork) ConnectionManager() ConnManager {
	return gsnet.host.ConnManager()
}

type tmGraphSyncNotifee tmGraphSyncNetwork

func (nn *tmGraphSyncNotifee) tmGraphSyncNetwork() *tmGraphSyncNetwork {
	return (*tmGraphSyncNetwork)(nn)
}

func (nn *tmGraphSyncNotifee) Connected(n network.Network, v network.Conn) {
	nn.tmGraphSyncNetwork().receiver.Connected(v.RemotePeer())
}

func (nn *tmGraphSyncNotifee) Disconnected(n network.Network, v network.Conn) {
	nn.tmGraphSyncNetwork().receiver.Disconnected(v.RemotePeer())
}

func (nn *tmGraphSyncNotifee) OpenedStream(n network.Network, v *rpc.Client) {}
func (nn *tmGraphSyncNotifee) ClosedStream(n network.Network, v *rpc.Client) {}
func (nn *tmGraphSyncNotifee) Listen(n network.Network, a ma.Multiaddr)      {}
func (nn *tmGraphSyncNotifee) ListenClose(n network.Network, a ma.Multiaddr) {}
