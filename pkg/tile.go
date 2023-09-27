package pkg

import (
	"errors"
	"image"
	"image/color"
	"image/draw"
)

const (
	inputIntW = 30
)

const (
	gridMaxWidth    = 160
	gridMaxHeight   = 80
	gridDivisionsXY = 5
	subtileHeight   = gridMaxHeight / gridDivisionsXY
	subtileWidth    = gridMaxWidth / gridDivisionsXY
	halfTileW       = subtileWidth >> 1
	halfTileH       = subtileHeight >> 1
)

// Tile is a representation of a map tile
type Tile struct {
	dt1                *DT1
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
}

func (t *Tile) Image() image.Image {
	floorPix, wallPix := t.makePixelBuffer()
	if len(floorPix) == 0 || len(wallPix) == 0 {
		return nil
	}

	tw, th := int(t.Width), int(t.Height)
	if th < 0 {
		th *= -1
	}

	rect := image.Rect(0, 0, tw, th)
	imgFloor, imgWall := image.NewRGBA(rect), image.NewRGBA(rect)
	imgFloor.Pix, imgWall.Pix = floorPix, wallPix

	return compositeImage(imgFloor, imgWall)
}

// Composite creates a new image by drawing src on top of dst.
func compositeImage(dst, src image.Image) *image.RGBA {
	// Initialize a blank RGBA image with the size of dst
	compositeImg := image.NewRGBA(dst.Bounds())

	// Draw dst onto the new image
	draw.Draw(compositeImg, compositeImg.Bounds(), dst, dst.Bounds().Min, draw.Src)

	// Draw src on top of the new image
	draw.Draw(compositeImg, src.Bounds(), src, src.Bounds().Min, draw.Over)

	return compositeImg
}

func bytesToImage(width, height int, data []byte) (*image.RGBA, error) {
	if len(data) != width*height*4 {
		return nil, errors.New("data length mismatch with width and height")
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	idx := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			alpha := data[idx]
			red := data[idx+1]
			green := data[idx+2]
			blue := data[idx+3]
			img.SetRGBA(x, y, color.RGBA{R: red, G: green, B: blue, A: alpha})
			idx += 4
		}
	}

	return img, nil
}

func (t *Tile) makePixelBuffer() (floorBuf, wallBuf []byte) {
	const (
		rOff = iota // rg,b offsets
		gOff
		bOff
		aOff
		bpp // bytes per pixel
	)

	tw, th := int(t.Width), int(t.Height)
	if th < 0 {
		th *= -1
	}

	var tileYMinimum int32

	for _, block := range t.Blocks {
		tileYMinimum = MinInt32(tileYMinimum, int32(block.Y))
	}

	tileYOffset := AbsInt32(tileYMinimum)

	floor := make([]byte, tw*th) // indices into palette
	wall := make([]byte, tw*th)  // indices into palette

	decodeTileGfxData(t.Blocks, &floor, &wall, tileYOffset, t.Width)

	floorBuf = make([]byte, tw*th*bpp)
	wallBuf = make([]byte, tw*th*bpp)

	for idx := range floor {
		var r, g, b, alpha byte

		floorVal := floor[idx]
		wallVal := wall[idx]

		rPos, gPos, bPos, aPos := idx*bpp+rOff, idx*bpp+gOff, idx*bpp+bOff, idx*bpp+aOff

		// the faux rgb color data here is just to make it look more interesting
		if t.dt1.palette != nil {
			col := t.dt1.palette[floorVal]
			r32, g32, b32, _ := col.RGBA()
			r, g, b = byte(r32), byte(g32), byte(b32)
		} else {
			r = floorVal
			g = floorVal
			b = floorVal
		}

		floorBuf[rPos] = r
		floorBuf[gPos] = g
		floorBuf[bPos] = b

		if floorVal > 0 {
			alpha = 255
		} else {
			alpha = 0
		}

		floorBuf[aPos] = alpha

		if t.dt1.palette != nil {
			col := t.dt1.palette[wallVal]
			r32, g32, b32, _ := col.RGBA()
			r, g, b = byte(r32), byte(g32), byte(b32)
		} else {
			r = wallVal
			g = wallVal
			b = wallVal
		}

		wallBuf[rPos] = r
		wallBuf[gPos] = g
		wallBuf[bPos] = b

		if wallVal > 0 {
			alpha = 255
		} else {
			alpha = 0
		}

		wallBuf[aPos] = alpha
	}

	return floorBuf, wallBuf
}

// we want to render the isometric (floor) and rle (wall) pixel buffers separately
func decodeTileGfxData(blocks []*Block, floorPixBuf, wallPixBuf *[]byte, tileYOffset, tileWidth int32) {
	for i := range blocks {
		switch blocks[i].Format() {
		case BlockFormatIsometric:
			DecodeTileGfxData([]*Block{blocks[i]}, floorPixBuf, tileYOffset, tileWidth)
		case BlockFormatRLE:
			DecodeTileGfxData([]*Block{blocks[i]}, wallPixBuf, tileYOffset, tileWidth)
		}
	}
}

const (
	blockDataLength = 256
)

// DecodeTileGfxData decodes tile graphics data for a slice of dt1 blocks
func DecodeTileGfxData(blocks []*Block, pixels *[]byte, tileYOffset, tileWidth int32) {
	for _, block := range blocks {
		if block.Format() == BlockFormatIsometric {
			// 3D isometric decoding
			xjump := []int32{14, 12, 10, 8, 6, 4, 2, 0, 2, 4, 6, 8, 10, 12, 14}
			nbpix := []int32{4, 8, 12, 16, 20, 24, 28, 32, 28, 24, 20, 16, 12, 8, 4}
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
					offset := ((blockY + y + tileYOffset) * tileWidth) + (blockX + x)
					(*pixels)[offset] = block.EncodedData[idx]
					x++
					n--
					idx++
				}
				y++
			}

			continue
		}
		// RLE Encoding
		blockX := int32(block.X)
		blockY := int32(block.Y)
		x := int32(0)
		y := int32(0)
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
				offset := ((blockY + y + tileYOffset) * tileWidth) + (blockX + x)
				(*pixels)[offset] = block.EncodedData[idx]
				idx++
				x++
				b2--
			}
		}
	}
}

// ImgIndexToRGBA converts the given indices byte slice and palette into
// a byte slice of RGBA values
func ImgIndexToRGBA(indexData []byte, palette color.Palette) []byte {
	bytesPerPixel := 4
	colorData := make([]byte, len(indexData)*bytesPerPixel)

	for i := 0; i < len(indexData); i++ {
		// Index zero is hardcoded transparent regardless of palette
		if indexData[i] == 0 {
			continue
		}

		c := palette[int(indexData[i])]

		r, g, b, a := c.RGBA()

		colorData[i*bytesPerPixel] = byte(r)
		colorData[i*bytesPerPixel+1] = byte(g)
		colorData[i*bytesPerPixel+2] = byte(b)
		colorData[i*bytesPerPixel+3] = byte(a)
	}

	return colorData
}

func MinInt32(a, b int32) int32 {
	if a < b {
		return a
	}

	return b
}

func AbsInt32(a int32) int32 {
	if a < 0 {
		return -a
	}

	return a
}
