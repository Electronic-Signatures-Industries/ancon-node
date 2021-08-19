package repository

import (
	"go-xdv-node/internal/metadata/app"
)

type IpldRepository struct{}

func (r *IpldRepository) Add(app.Metadata, ...interface{}) error {
	// 1 - validar tx
	// 2 - validar data
	// 3 - crear json como ipld
	// 4 - escribir al storage
	//ipld.Write()
	return nil
}
