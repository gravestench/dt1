package v2

import (
	"fmt"
	"image/color"
	"io"
	"math"

	"github.com/gravestench/bitstream"
)

func New(buffer io.Reader) (*DT1, error) {
	d := &DT1{}

	stream := bitstream.NewReader(buffer)

	if err := d.decodeHeader(stream); err != nil {
		return nil, fmt.Errorf("decoding header: %v", err)
	}

	if err := d.decodeBody(stream); err != nil {
		return nil, fmt.Errorf("decoding header: %v", err)
	}

	return d, nil
}

type DT1 struct {
	header struct {
		V1, V2 int32
	}

	Tiles   []*Tile
	palette color.Palette
}

func (d *DT1) Palette() color.Palette {
	return d.palette
}

func (d *DT1) SetPalette(p color.Palette) {
	d.palette = p
	for _, tile := range d.Tiles {
		tile.palette = p

		for _, block := range tile.Blocks {
			block.palette = p
		}
	}
}

func (d *DT1) decodeHeader(stream *bitstream.Reader) error {
	const (
		unknownDataBytes = 260
		numTileBytes     = 4
		dataAddressBytes = 4
	)

	if err := d.decodeDT1Version(stream); err != nil {
		return fmt.Errorf("decoding version: %v", err)
	}

	// we just skip these for now :shrug:
	if res := stream.Next(unknownDataBytes).Bytes(); res.Error != nil {
		return res.Error
	}

	numberOfTiles, err := stream.Next(numTileBytes).Bytes().AsInt32()
	if err != nil {
		return fmt.Errorf("decoding number of tiles: %v", err)
	}

	tileDataStartAddress, err := stream.Next(dataAddressBytes).Bytes().AsInt32()
	if err != nil {
		return fmt.Errorf("decoding tile data start address: %v", err)
	}

	stream.SetPosition(int(tileDataStartAddress))

	d.Tiles = make([]*Tile, numberOfTiles)

	return nil
}

func (d *DT1) decodeDT1Version(stream *bitstream.Reader) error {
	const (
		v1Len, v2Len           = 4, 4
		expectedV1, expectedV2 = 7, 6
	)

	ver1, _ := stream.Next(v1Len).Bytes().AsInt32()
	ver2, _ := stream.Next(v2Len).Bytes().AsInt32()

	if ver1 != expectedV1 || ver2 != expectedV2 {
		return fmt.Errorf("expected to have a version of %d.%d, got %d.%d instead", expectedV1, expectedV2, ver1, ver2)
	}

	d.header.V1 = ver1
	d.header.V2 = ver2

	return nil
}

func (d *DT1) decodeBody(stream *bitstream.Reader) error {
	if err := d.decodeTileHeaders(stream); err != nil {
		return fmt.Errorf("decoding stage 1: %v", err)
	}

	if err := d.decodeTileBodies(stream); err != nil {
		return fmt.Errorf("decoding stage 2: %v", err)
	}

	return nil
}

func (d *DT1) decodeTileHeaders(stream *bitstream.Reader) error {
	const (
		directionBytes  = 4
		roofHeightBytes = 2
		materialsBytes  = 2
		tileHeightBytes = 4
		tileWidthBytes  = tileHeightBytes

		tileTypeBytes        = 4
		tileStyleBytes       = 4
		tileSequenceBytes    = 4
		tileRarityIndexBytes = 4

		tileBlockHeaderPointerBytes = 4
		tileBlockHeaderSizeBytes    = 4
		tileNumBlocksBytes          = 4

		unknownData1Bytes = 4
		unknownData2Bytes = 4
		unknownData3Bytes = 7
		unknownData4Bytes = 12
	)

	// for brevity, we will throw away errors from bitstream until last error
	for tileIdx := range d.Tiles {
		tile := &Tile{}

		tile.Direction, _ = stream.Next(directionBytes).Bytes().AsInt32()
		tile.RoofHeight, _ = stream.Next(roofHeightBytes).Bytes().AsInt16()

		materials, _ := stream.Next(materialsBytes).Bytes().AsUInt16()
		tile.MaterialFlags = NewMaterialFlags(materials)

		tile.Height, _ = stream.Next(tileHeightBytes).Bytes().AsInt32()
		tile.Width, _ = stream.Next(tileWidthBytes).Bytes().AsInt32()

		stream.Next(unknownData1Bytes).Bytes() // skip

		tile.Type, _ = stream.Next(tileTypeBytes).Bytes().AsInt32()
		tile.Style, _ = stream.Next(tileStyleBytes).Bytes().AsInt32()
		tile.Sequence, _ = stream.Next(tileSequenceBytes).Bytes().AsInt32()
		tile.RarityFrameIndex, _ = stream.Next(tileRarityIndexBytes).Bytes().AsInt32()

		stream.Next(unknownData2Bytes).Bytes() // skip

		for i := range tile.SubTileFlags {
			subtileFlag, _ := stream.Next(1).Bytes().AsByte()
			tile.SubTileFlags[i] = NewSubTileFlags(subtileFlag)
		}

		stream.Next(unknownData3Bytes).Bytes() // skip

		tile.blockHeaderPointer, _ = stream.Next(tileBlockHeaderPointerBytes).Bytes().AsInt32()
		tile.blockHeaderSize, _ = stream.Next(tileBlockHeaderSizeBytes).Bytes().AsInt32()
		numBlocks, _ := stream.Next(tileNumBlocksBytes).Bytes().AsInt32()
		tile.Blocks = make([]*Block, numBlocks)

		if err := stream.Next(unknownData4Bytes).Bytes().Error; err != nil {
			return fmt.Errorf("skipping data bytes: %v", err)
		}

		d.Tiles[tileIdx] = tile
	}

	return nil
}

func (d *DT1) decodeTileBodies(stream *bitstream.Reader) error {
	for idx := range d.Tiles {
		if err := d.Tiles[idx].decodeBlockHeaders(stream); err != nil {
			return fmt.Errorf("decoding black headers: %v", err)
		}

		if err := d.Tiles[idx].decodeBlockBodies(stream); err != nil {
			return fmt.Errorf("decoding block bodies: %v", err)
		}
	}

	return nil
}

// DecodeTileGfxData decodes tile graphics data for a slice of dt1 blocks
func (d *DT1) decodeTileGraphics() {
	// Determine the Y offset which will be used in decoding process
	yOffset := d.determineYOffset()

	// Loop through each tile in the DT1
	for _, tile := range d.Tiles {
		tileWidth := tile.Width
		tileHeight := tile.Height

		// Handle negative height
		if tileHeight < 0 {
			tileHeight *= -1
		}

		// Loop through each block in the tile
		for _, block := range tile.Blocks {
			// Initialize pixel data for the block based on tile dimensions
			block.PixelData = make([]byte, tileWidth*tileHeight)

			// Decode the block graphics based on its format
			switch block.format {
			case BlockEncodingIsometric:
				block.decodeIsometric(tileWidth, yOffset)
			case BlockEncodingRLE:
				block.decodeRunLengthEncoded(tileWidth, yOffset)
			}

			block.createAndAssignRGBAImage(int(tileWidth), int(tileHeight))
		}
	}
}

func (d *DT1) determineYOffset() int32 {
	// Initialize the minimumYOffset with a high positive value for comparison.
	minimumYOffset := int32(math.MaxInt32)

	for _, tile := range d.Tiles {
		for _, block := range tile.Blocks {
			// Update the minimumYOffset if the current block's Y is smaller.
			if int32(block.Y) < minimumYOffset {
				minimumYOffset = int32(block.Y)
			}
		}
	}

	// Return the smallest Y value (which can be negative).
	return minimumYOffset
}

// Helper function to return the maximum of two int32 values
func max(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}
