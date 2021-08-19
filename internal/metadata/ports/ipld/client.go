package ipld

import (
	"bytes"
	"context"
	"io"

	"github.com/ipfs/go-cid"

	cidlink "github.com/ipld/go-ipld-prime/linking/cid"

	ipld "github.com/ipld/go-ipld-prime"
	"github.com/ipld/go-ipld-prime/schema"
)

func Wite(ctx context.Context, n schema.TypedNode, prefix cid.Prefix) ipld.Link {
	lsys := cidlink.DefaultLinkSystem()

	lsys.StorageWriteOpener = func(lnkCtx ipld.LinkContext) (io.Writer, ipld.BlockWriteCommitter, error) {
		buf := bytes.Buffer{}
		return &buf, func(lnk ipld.Link) error {
			//TODO execute smart contract
			//TODO use a discriminator here
			return nil
		}, nil
	}

	// Add Documen

	// prefix := x{cid.Prefi
	// 	Version:  1,
	// 	Codec:    0x71, // dag-cbor
	// 	MhType:   0x13, // sha2-512
	// 	MhLength: 64,   // sha2-512 hash has a 64-byte sum.
	// }}

	lp := cidlink.LinkPrototype{prefix}

	lnk, err := lsys.Store(
		ipld.LinkContext{}, // The zero value is fine.  Configure it it you want cancellability or other features.
		lp,                 // The LinkPrototype says what codec and hashing to use.
		n,                  // And here's our data.
	)
	if err != nil {
		panic(err)
	}

	return lnk
}
