package v2

import (
	"image"
	"image/color"
)

// Block represents a DT1 block
type Block struct {
	X           int16
	Y           int16
	GridX       byte
	GridY       byte
	format      BlockEncoding
	EncodedData []byte
	Length      int32
	FileOffset  int32
	PixelData   []byte
	palette     color.Palette
	image       *image.RGBA
}

func (block *Block) ColorModel() color.Model {
	return block.palette
}

func (block *Block) Bounds() image.Rectangle {
	if block.image != nil {
		return block.image.Bounds()
	}

	return image.Rect(0, 0, 0, 0)
}

func (block *Block) At(x, y int) color.Color {
	if index := y*block.Bounds().Dx() + x; index < len(block.PixelData) {
		return block.palette[block.PixelData[index]]
	}

	return color.Transparent // return a transparent color if index out of bounds
}

func (block *Block) ColorIndexAt(x, y int) uint8 {
	if index := y*block.Bounds().Dx() + x; index < len(block.PixelData) {
		return block.PixelData[index]
	}

	return 0 // return a transparent color if index out of bounds
}

func (block *Block) decodeIsometric(imageWidth, verticalOffset int32) {
	const blockDataLength = 256

	// These arrays define the starting X position and the number of pixels
	// to process for each row in an isometric block.
	startXPositions := []int32{14, 12, 10, 8, 6, 4, 2, 0, 2, 4, 6, 8, 10, 12, 14}
	pixelsPerRow := []int32{4, 8, 12, 16, 20, 24, 28, 32, 28, 24, 20, 16, 12, 8, 4}

	blockStartX := int32(block.X)
	blockStartY := int32(block.Y)
	remainingDataLength := int32(blockDataLength)
	dataCursor := 0
	currentY := int32(0)

	for remainingDataLength > 0 {
		// Ensure we don't exceed the arrays' boundaries
		if currentY >= int32(len(startXPositions)) || currentY >= int32(len(pixelsPerRow)) {
			break
		}

		currentX := startXPositions[currentY]
		pixelsToProcess := pixelsPerRow[currentY]

		// Ensure we don't try to decode more data than what's available
		if dataCursor+int(pixelsToProcess) > len(block.EncodedData) {
			break
		}

		remainingDataLength -= pixelsToProcess

		for pixelsToProcess > 0 {
			pixelDataOffset := ((blockStartY + currentY + verticalOffset) * imageWidth) + (blockStartX + currentX)

			// Ensure we don't write beyond PixelData's boundary
			if pixelDataOffset >= int32(len(block.PixelData)) {
				break
			}

			block.PixelData[pixelDataOffset] = block.EncodedData[dataCursor]

			dataCursor++
			currentX++
			pixelsToProcess--
		}

		currentY++
	}
}

func (block *Block) decodeRunLengthEncoded(imageWidth, verticalOffset int32) {
	// Convert block's top-left corner coordinates to int32
	blockStartX := int32(block.X)
	blockStartY := int32(block.Y)

	// Initialize cursor position within the EncodedData array
	dataCursor := 0
	remainingDataLength := block.Length

	// Initial position within the block
	currentX, currentY := int32(0), int32(0)

	// Process encoded data while there's remaining data to decode
	for remainingDataLength > 0 {
		// Check if we have at least two bytes left for runStartOffset and runLength
		if dataCursor+1 >= len(block.EncodedData) {
			break
		}

		// Extract the next two bytes which represent the run-length encoding of the pixel data
		runStartOffset := block.EncodedData[dataCursor]
		runLength := block.EncodedData[dataCursor+1]
		dataCursor += 2
		remainingDataLength -= 2

		// If both bytes are zero, it signifies the end of the current row
		if (runStartOffset | runLength) == 0 {
			currentX = 0
			currentY++
			continue
		}

		// Adjust the current X position
		currentX += int32(runStartOffset)

		// Check if we have enough bytes left for the encoded data
		if dataCursor+int(runLength) > len(block.EncodedData) {
			break
		}

		// Deduct the runLength from remainingDataLength, ensuring it doesn't go negative
		remainingDataLength = max(0, remainingDataLength-int32(runLength))

		// Decode the run-length encoded data
		for runLength > 0 {
			// Check if we've crossed the image width (assuming wrap-around behavior)
			if currentX >= imageWidth {
				currentX = 0
				currentY++
			}

			// Calculate the position in the PixelData array
			pixelDataOffset := ((blockStartY + currentY + verticalOffset) * imageWidth) + (blockStartX + currentX)

			// Boundary check for PixelData array
			if pixelDataOffset >= int32(len(block.PixelData)) {
				break
			}

			// Copy the pixel value to the PixelData array
			block.PixelData[pixelDataOffset] = block.EncodedData[dataCursor]

			// Move to the next pixel and encoded data byte
			dataCursor++
			currentX++
			runLength--
		}
	}
}

func (block *Block) createAndAssignRGBAImage(tileWidth, tileHeight int) image.Image {
	// Calculate the total number of pixels based on the tile's dimensions.
	numPixels := tileWidth * tileHeight

	// Initialize the image with the specified dimensions.
	img := image.NewRGBA(image.Rect(0, 0, tileWidth, tileHeight))

	// Iterate over all pixels.
	for i := 0; i < numPixels; i++ {
		// Calculate the start index for this pixel in the PixelData slice.
		startIdx := i * 4 // Since each pixel has 4 bytes (R, G, B, A)

		// Make sure we won't go out of bounds in the PixelData slice.
		if startIdx+4 > len(block.PixelData) {
			break
		}

		// Extract the R, G, B, and A values.
		r, g, b, a := block.PixelData[startIdx], block.PixelData[startIdx+1], block.PixelData[startIdx+2], block.PixelData[startIdx+3]

		// Set the pixel in the image.
		x, y := i%tileWidth, i/tileWidth
		img.Set(x, y, color.RGBA{r, g, b, a})
	}

	// Assign the image to the block's image field.
	return img
}

// we want to render the isometric (floor) and rle (wall) pixel buffers separately
func (t *Tile) decodeTileGfxData(tileYOffset, tileWidth int32) {
	for _, block := range t.Blocks {
		switch block.format {
		case BlockEncodingIsometric:
			block.decodeIsometric(tileYOffset, tileWidth)
		case BlockEncodingRLE:
			block.decodeRunLengthEncoded(tileYOffset, tileWidth)
		}
	}
}
