package ancon

import (
	"log"
	"sync"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/tharsis/ethermint/ethereum/rpc/namespaces/eth"
	"go.etcd.io/etcd/mvcc/backend"

	rpcclient "github.com/tendermint/tendermint/rpc/jsonrpc/client"
)

type API struct {
	ctx       *server.Context
	logger    log.Logger
	clientCtx client.Context
	backend   backend.Backend
}

type AddrLocker struct {
	mu    sync.Mutex
	locks map[common.Address]*sync.Mutex
}

// NewMinerAPI creates an instance of the Miner API.
func NewMinerAPI(
	ctx *server.Context,
	clientCtx client.Context,
	backend backend.Backend,
) *API {
	return &API{
		ctx:       ctx,
		clientCtx: clientCtx,
		logger:    ctx.Logger.With("api", "miner"),
		backend:   backend,
	}
}

// SetEtherbase sets the etherbase of the miner
func (api *API) SetEtherbase(etherbase common.Address) bool {

	return true
}

// SetGasPrice sets the minimum accepted gas price for the miner.
// NOTE: this function accepts only integers to have the same interface than go-eth
// to use float values, the gas prices must be configured using the configuration file
func (api *API) SetGasPrice(gasPrice hexutil.Big) bool {

	return true
}

func test(ctx *server.Context, clientCtx client.Context, tmWSClient *rpcclient.WSClient, selectedAPIs []string) {
	evmBackend := backend.NewEVMBackend(ctx, ctx.Logger, clientCtx)
	nonceLock := new(AddrLocker)

	_ = rpc.API{
		Namespace: "",
		Version:   1,
		Service:   eth.NewPublicAPI(ctx.Logger, clientCtx, evmBackend, nonceLock),
		Public:    true,
	}
}
