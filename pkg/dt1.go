package pkg

import (
	"bytes"
	"fmt"
	"image/color"
	"math"

	"github.com/gravestench/bitstream"
)

// FromBytes loads a DT1 record
func FromBytes(fileData []byte) (result *DT1, err error) {
	result = &DT1{}
	stream := bitstream.NewReader(bytes.NewReader(fileData))

	if err = result.decodeDT1Header(stream); err != nil {
		return nil, err
	}

	if err = result.decodeDT1Body(stream); err != nil {
		return nil, err
	}

	return result, nil
}

// DT1 represents a DT1 file.
type DT1 struct {
	Tiles   []*Tile
	palette color.Palette
}

// BlockDataFormat represents the format of the block data
type BlockDataFormat int16

const (
	// BlockFormatRLE specifies the block format is RLE encoded
	BlockFormatRLE BlockDataFormat = 0

	// BlockFormatIsometric specifies the block format isometrically encoded
	BlockFormatIsometric BlockDataFormat = 1
)

func (d *DT1) decodeDT1Header(stream *bitstream.Reader) error {
	const (
		unknownDataBytes = 260
		numTileBytes     = 4
		dataAddressBytes = 4
	)

	if err := d.decodeDT1Version(stream); err != nil {
		return err
	}

	// we just skip these for now :shrug:
	if res := stream.Next(unknownDataBytes).Bytes(); res.Error != nil {
		return res.Error
	}

	numberOfTiles, err := stream.Next(numTileBytes).Bytes().AsInt32()
	if err != nil {
		return err
	}

	tileDataStartAddress, err := stream.Next(dataAddressBytes).Bytes().AsInt32()
	if err != nil {
		return err
	}

	stream.SetPosition(int(tileDataStartAddress))

	d.Tiles = make([]*Tile, numberOfTiles)

	return nil
}

func (d *DT1) decodeDT1Body(stream *bitstream.Reader) error {
	if err := d.decodeTilesStage1(stream); err != nil {
		return err
	}

	if err := d.decodeTilesStage2(stream); err != nil {
		return err
	}

	return nil
}

func (d *DT1) decodeDT1Version(stream *bitstream.Reader) error {
	const (
		v1Bytes, v2Bytes       = 4, 4
		expectedV1, expectedV2 = 7, 6
	)

	ver1, _ := stream.Next(v1Bytes).Bytes().AsInt32()
	ver2, _ := stream.Next(v2Bytes).Bytes().AsInt32()
	if ver1 != expectedV1 || ver2 != expectedV2 {
		const fmtErr = "expected to have a version of %d.%d, got %d.%d instead"
		return fmt.Errorf(fmtErr, expectedV1, expectedV2, ver1, ver2)
	}

	return nil
}

func (d *DT1) decodeTilesStage1(stream *bitstream.Reader) error {
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
		newTile := &Tile{
			dt1: d,
		}

		newTile.Direction, _ = stream.Next(directionBytes).Bytes().AsInt32()
		newTile.RoofHeight, _ = stream.Next(roofHeightBytes).Bytes().AsInt16()

		materials, _ := stream.Next(materialsBytes).Bytes().AsUInt16()
		newTile.MaterialFlags = NewMaterialFlags(materials)

		newTile.Height, _ = stream.Next(tileHeightBytes).Bytes().AsInt32()
		newTile.Width, _ = stream.Next(tileWidthBytes).Bytes().AsInt32()

		stream.Next(unknownData1Bytes).Bytes() // skip

		newTile.Type, _ = stream.Next(tileTypeBytes).Bytes().AsInt32()
		newTile.Style, _ = stream.Next(tileStyleBytes).Bytes().AsInt32()
		newTile.Sequence, _ = stream.Next(tileSequenceBytes).Bytes().AsInt32()
		newTile.RarityFrameIndex, _ = stream.Next(tileRarityIndexBytes).Bytes().AsInt32()

		stream.Next(unknownData2Bytes).Bytes() // skip

		for i := range newTile.SubTileFlags {
			subtileFlag, _ := stream.Next(1).Bytes().AsByte()
			newTile.SubTileFlags[i] = NewSubTileFlags(subtileFlag)
		}

		stream.Next(unknownData3Bytes).Bytes() // skip

		newTile.blockHeaderPointer, _ = stream.Next(tileBlockHeaderPointerBytes).Bytes().AsInt32()
		newTile.blockHeaderSize, _ = stream.Next(tileBlockHeaderSizeBytes).Bytes().AsInt32()
		numBlocks, _ := stream.Next(tileNumBlocksBytes).Bytes().AsInt32()
		newTile.Blocks = make([]*Block, numBlocks)

		err := stream.Next(unknownData4Bytes).Bytes().Error // skip, check error
		if err != nil {
			return err
		}

		d.Tiles[tileIdx] = newTile
	}

	return nil
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
		if t.Blocks[blockIdx] == nil {
			t.Blocks[blockIdx] = &Block{tile: t}
		}

		block := t.Blocks[blockIdx]

		block.X, _ = stream.Next(blockXYBytes).Bytes().AsInt16()
		block.Y, _ = stream.Next(blockXYBytes).Bytes().AsInt16()

		stream.Next(blockUnknown1Bytes).Bytes()

		block.GridX, _ = stream.Next(blockGridXYBytes).Bytes().AsByte()
		block.GridY, _ = stream.Next(blockGridXYBytes).Bytes().AsByte()

		formatValue, _ := stream.Next(blockFormatValueBytes).Bytes().AsInt16()

		block.Format = BlockFormatRLE

		if formatValue == 1 {
			block.Format = BlockFormatIsometric
		}

		block.Length, _ = stream.Next(blockLengthBytes).Bytes().AsInt32()

		stream.Next(blockUnknown2Bytes).Bytes()

		block.FileOffset, err = stream.Next(blockFileOffsetBytes).Bytes().AsInt32()
		if err != nil {
			return err
		}
	}

	return nil
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

func (d *DT1) decodeTilesStage2(stream *bitstream.Reader) error {
	for tileIdx := range d.Tiles {
		if err := d.Tiles[tileIdx].decodeBlockHeaders(stream); err != nil {
			return err
		}

		if err := d.Tiles[tileIdx].decodeBlockBodies(stream); err != nil {
			return err
		}
	}

	return nil
}

func (d *DT1) Palette() color.Palette {
	if d.palette == nil {
		d.palette = defaultPalette()
	}

	return d.palette
}

func defaultPalette() color.Palette {
	const numColors = 256

	palette := make(color.Palette, numColors)

	for idx := range palette {
		palette[idx] = color.RGBA{
			R: uint8(idx),
			G: uint8(idx),
			B: uint8(idx),
			A: math.MaxUint8,
		}
	}

	return palette
}
