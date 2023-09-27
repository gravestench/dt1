package pkg

import (
	"image"
	"image/color"
)

var _ image.PalettedImage = &Block{}

// Block represents a DT1 block
type Block struct {
	tile        *Tile
	X           int16
	Y           int16
	GridX       byte
	GridY       byte
	format      BlockDataFormat
	EncodedData []byte
	Length      int32
	FileOffset  int32
	PixelData   []byte
	image       *image.RGBA
}

func (block *Block) ColorIndexAt(x, y int) uint8 {
	w := block.image.Bounds().Dx()
	absIdx := (y * w) + x

	if absIdx < 0 {
		absIdx = 0
	}

	if absIdx >= len(block.PixelData) {
		return 0
	}

	return block.PixelData[absIdx]
}

func (block *Block) ColorModel() color.Model {
	return color.RGBAModel
}

func (block *Block) Bounds() image.Rectangle {
	return block.image.Bounds()
}

func (block *Block) At(x, y int) color.Color {
	palIdx := block.ColorIndexAt(x, y)
	pal := block.tile.dt1.Palette()

	return pal[palIdx]
}

// Format returns block format
func (b *Block) Format() BlockDataFormat {
	if b.format == 1 {
		return BlockFormatIsometric
	}

	return BlockFormatRLE
}
