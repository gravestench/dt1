package pkg

import "image"

// DecodeTileGfxData decodes tile graphics data for a slice of dt1 blocks
func (d *DT1) decodeTileGraphics() {
	yOffset := d.determineYOffset()

	for _, tile := range d.Tiles {
		tw, th := tile.Width, tile.Height
		if th < 0 {
			th *= -1
		}

		for _, block := range tile.Blocks {
			block.PixelData = make([]byte, tw*th)

			if block.format == BlockFormatIsometric {
				block.decodeIsometric(tw, yOffset)

				continue
			}

			block.decodeRunLengthEncoded(tw, yOffset)

			block.image = image.NewRGBA(image.Rect(0, 0, int(tw), int(th)))
		}
	}
}

func (d *DT1) determineYOffset() (yOffset int32) {
	for _, tile := range d.Tiles {
		for blockIdx := range tile.Blocks {
			block := tile.Blocks[blockIdx]

			if int32(block.Y) < yOffset {
				yOffset = int32(block.Y)
			}
		}

		if yOffset < 0 {
			yOffset *= -1
		}
	}

	return yOffset
}

/*
the way the data is encoded is in runs of non-blank pixels
in the following diagram, an `x` is an opaque pixel

	  xjump -------|
		           |
		           v
		            xxxx <----- nbpix[0] == 4
		          xxxxxxxx <----- nbpix[1] == 8
		        xxxxxxxxxxxx
		      xxxxxxxxxxxxxxxx
		    xxxxxxxxxxxxxxxxxxxx
		  xxxxxxxxxxxxxxxxxxxxxxxx
		xxxxxxxxxxxxxxxxxxxxxxxxxxxx
	  xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
		xxxxxxxxxxxxxxxxxxxxxxxxxxxx
		  xxxxxxxxxxxxxxxxxxxxxxxx
		    xxxxxxxxxxxxxxxxxxxx
		      xxxxxxxxxxxxxxxx
		        xxxxxxxxxxxx
		          xxxxxxxx
		            xxxx

`xjump` is the number of pixels from the left edge, for each group of non-blank pixels
the index into xjump is the current row

`nbpix` contains the integer length for runs of non-blank pixels
the index into nbpix is the current row

the encoding relies on the 0-value being encoded as the default palette index
for each pixel, which is always(?) the transparent color of the palette being used
*/
func (block *Block) decodeIsometric(w, yOffset int32) {
	const (
		blockDataLength = 256
	)

	xjump := []int32{14, 12, 10, 8, 6, 4, 2, 0, 2, 4, 6, 8, 10, 12, 14}
	nbpix := []int32{4, 8, 12, 16, 20, 24, 28, 32, 28, 24, 20, 16, 12, 8, 4}

	// 3D isometric decoding
	blockX := int32(block.X)
	blockY := int32(block.Y)
	length := int32(blockDataLength)
	x := int32(0)
	y := int32(0)
	idx := 0

	for length > 0 {
		x = xjump[y]
		n := nbpix[y]
		length -= n

		for n > 0 {
			offset := ((blockY + y + yOffset) * w) + (blockX + x)
			block.PixelData[offset] = block.EncodedData[idx]
			x++
			n--
			idx++
		}
		y++
	}
}

func (block *Block) decodeRunLengthEncoded(w, yOffset int32) {
	var blockX, blockY, x, y int32

	blockX = int32(block.X)
	blockY = int32(block.Y)

	idx := 0
	length := block.Length

	for length > 0 {
		b1 := block.EncodedData[idx]
		b2 := block.EncodedData[idx+1]
		idx += 2
		length -= 2

		if (b1 | b2) == 0 {
			x = 0
			y++

			continue
		}

		x += int32(b1)
		length -= int32(b2)

		for b2 > 0 {
			offset := ((blockY + y + yOffset) * w) + (blockX + x)
			block.PixelData[offset] = block.EncodedData[idx]
			idx++
			x++
			b2--
		}
	}
}
