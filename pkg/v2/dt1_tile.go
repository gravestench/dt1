package v2

import (
	"image"
	"image/color"

	"github.com/gravestench/bitstream"
)

// Tile is a representation of a map tile
type Tile struct {
	Direction          int32
	RoofHeight         int16
	MaterialFlags      MaterialFlags
	Height             int32
	Width              int32
	Type               int32
	Style              int32
	Sequence           int32
	RarityFrameIndex   int32
	SubTileFlags       [25]SubTileFlags
	blockHeaderPointer int32
	blockHeaderSize    int32
	Blocks             []*Block
	palette            color.Palette
	image              struct {
		floor *image.RGBA
		wall  *image.RGBA
	}
}

func (t *Tile) decodeBlockHeaders(stream *bitstream.Reader) (err error) {
	const (
		blockXYBytes          = 2
		blockGridXYBytes      = 1
		blockFormatValueBytes = 2
		blockLengthBytes      = 4
		blockFileOffsetBytes  = 4

		blockUnknown1Bytes = 2
		blockUnknown2Bytes = 2
	)

	stream.SetPosition(int(t.blockHeaderPointer))

	for blockIdx := range t.Blocks {
		block := &Block{}

		block.X, _ = stream.Next(blockXYBytes).Bytes().AsInt16()
		block.Y, _ = stream.Next(blockXYBytes).Bytes().AsInt16()

		stream.Next(blockUnknown1Bytes).Bytes() // skip

		block.GridX, _ = stream.Next(blockGridXYBytes).Bytes().AsByte()
		block.GridY, _ = stream.Next(blockGridXYBytes).Bytes().AsByte()

		formatValue, _ := stream.Next(blockFormatValueBytes).Bytes().AsInt16()
		block.format = BlockEncoding(formatValue)
		block.Length, _ = stream.Next(blockLengthBytes).Bytes().AsInt32()

		stream.Next(blockUnknown2Bytes).Bytes() // skip

		if block.FileOffset, err = stream.Next(blockFileOffsetBytes).Bytes().AsInt32(); err != nil {
			return err
		}

		t.Blocks[blockIdx] = block
	}

	return nil
}

func (t *Tile) FloorImage() image.Image {
	return t.image.floor
}

func (t *Tile) WallImage() image.Image {
	return t.image.wall
}

func (t *Tile) decodeBlockBodies(stream *bitstream.Reader) error {
	for blockIndex, block := range t.Blocks {
		stream.SetPosition(int(t.blockHeaderPointer + block.FileOffset))

		encodedData, err := stream.Next(int(block.Length)).Bytes().AsBytes()
		if err != nil {
			return err
		}

		t.Blocks[blockIndex].EncodedData = encodedData
	}

	return nil
}

func (t *Tile) ColorModel() color.Model {
	return t.palette
}

func (t *Tile) Bounds() image.Rectangle {
	return image.Rect(0, 0, int(t.Width), int(t.Height))
}

func (t *Tile) ColorIndexAt(x, y int) uint8 {
	for _, block := range t.Blocks {
		blockX, blockY := int(block.X), int(block.Y)

		// Check if (x, y) falls within this block's dimensions
		if x >= blockX && x < blockX+int(t.Width) && y >= blockY && y < blockY+int(t.Height) {
			// Translate the coordinates to be relative to the block's top-left corner
			relX, relY := x-blockX, y-blockY
			return block.ColorIndexAt(relX, relY)
		}
	}
	return 0 // default palette index
}

func (t *Tile) At(x, y int) color.Color {
	for _, block := range t.Blocks {
		blockX, blockY := int(block.X), int(block.Y)

		// Check if (x, y) falls within this block's dimensions
		if x >= blockX && x < blockX+int(t.Width) && y >= blockY && y < blockY+int(t.Height) {
			// Translate the coordinates to be relative to the block's top-left corner
			relX, relY := x-blockX, y-blockY
			return block.At(relX, relY)
		}
	}
	return color.RGBA{} // default color (transparent black)
}
