package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/dgraph-io/badger/v2"

	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/light"
	"github.com/tendermint/tendermint/light/provider"
	httpprovider "github.com/tendermint/tendermint/light/provider/http"
	dbs "github.com/tendermint/tendermint/light/store/db"
)

var configFile string

func init() {
	flag.StringVar(&configFile, "config", "$HOME/.ancon-protocold/config/config.toml", "Path to config.toml")
}

func newLightTendermint(ctx context.Context, chainID string,
	primary string, witness string, height int, hash []byte, db badger.DB) (*light.Client, error) {

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
		dbs.New(db),
		light.Logger(log.TestingLogger()),
	)
	//_, err := c.Update(ctx, time.Now())

	return c, nil
}

// Manually getting light blocks and verifying them.
func main() {

	ctx := context.Background()
	db, err := badger.Open(badger.DefaultOptions("/tmp/badger"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open badger db: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	flag.Parse()

	h := 20

	hash := []byte("")

	node, err := newLightTendermint(ctx, "", "http://localhost:26657", "http://localhost:26657", h, hash, db)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(2)
	}

	// rpc
}
