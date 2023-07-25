package dt1

import (
	"github.com/gravestench/dt1/pkg"
)

type (
	DT1             = pkg.DT1
	Tile            = pkg.Tile
	Block           = pkg.Block
	MaterialFlags   = pkg.MaterialFlags
	SubTileFlags    = pkg.SubTileFlags
	BlockDataFormat = pkg.BlockDataFormat
)

func FromBytes(fileData []byte) (result *DT1, err error) {
	return pkg.FromBytes(fileData)
}

func NewSubTileFlags(data byte) SubTileFlags {
	return pkg.NewSubTileFlags(data)
}

func NewMaterialFlags(data uint16) MaterialFlags {
	return pkg.NewMaterialFlags(data)
}
