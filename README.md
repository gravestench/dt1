<!-- PROJECT LOGO -->
<h1 align="center">DT1</h1>
<p align="center">
  Package for transcoding DT1 Tile files used in Diablo 2.
  <br />
  <br />
  <a href="https://github.com/gravestench/dt1/issues">Report Bug</a>
  ·
  <a href="https://github.com/gravestench/dt1/issues">Request Feature</a>
</p>

<!-- ABOUT THE PROJECT -->
## About

The DT1 Transcoder package provides a Go implementation for handling DT1 files,
which are used in Diablo 2 for representing tilesets.
This package allows you to read and decode DT1 files efficiently, providing
access to individual tiles and their properties.

## Getting Started

### Prerequisites
To use this DT1 transcoder package, ensure you have Go 1.16 or a later version
installed, and your Go environment is set up correctly.

### Installation
To install the package, you can use Go's standard `go get` command:

```shell
go get -u github.com/gravestench/dt1
```

### Usage
Once you have installed the package, you can use it in your Go applications by
importing it as follows:

```golang
import "github.com/gravestench/dt1"
```

#### Load DT1 File
To load a DT1 file from a byte slice, use the `FromBytes` function:

```golang
fileData := // Load your DT1 file data here as a byte slice
dt1, err := dt1.FromBytes(fileData)
if err != nil {
    // Handle error
}
// Use the dt1 object to access the DT1 file data
```

#### Accessing Tiles
After loading a DT1 file, you can access individual tiles and their properties:

```golang
// Access the tiles slice within the DT1 object
for _, tile := range dt1.Tiles {
    // Access tile properties
    direction := tile.Direction
    roofHeight := tile.RoofHeight
    materialFlags := tile.MaterialFlags
    height := tile.Height
    width := tile.Width
    // ...and other tile properties

    // Access blocks within the tile
    for _, block := range tile.Blocks {
        // Access block properties
        x := block.X
        y := block.Y
        gridX := block.GridX
        gridY := block.GridY
        format := block.Format
        length := block.Length
        encodedData := block.EncodedData
        // ...and other block properties
    }
}
```

### Features
The DT1 transcoder package offers the following features:
- Efficiently read and parse DT1 image files.
- Extract information about DT1 version, tile properties, and block data.
- Access individual tiles and their properties, as well as individual blocks
  within tiles.
- Provides a default grayscale palette for rendering DT1 images.

<!-- CONTRIBUTING -->
## Contributing

Contributions to the DT1 transcoder package are welcome and encouraged. If you
find any issues or have improvements to suggest, feel free to open an issue or
submit a pull request.

To contribute to the project, follow these steps:

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request