// api.go defines the interfaces and shared types for distributed Mandelbrot rendering.
// It is used by both server and client code to ensure consistent API contracts.

package mandel

import (
	"image"
	"time"
)

// Calling 'go generate api.go' will call 'irpc api.go', regenerating api_irpc.go code.
//go:generate irpc $GOFILE

// ImgProvider is implemented by the server and called by the CLI client to get the full image once rendering is complete.
type ImgProvider interface {
	// GetImage returns the fully rendered image.
	// Returns: image.RGBA pointer and error if retrieval fails.
	GetImage() (*image.RGBA, error)
}

// TileProvider is implemented by the server and used by the web client to show rendering progress tile by tile.
// Web clients use polling to check for updates.
type TileProvider interface {
	// FinishedTiles returns a map of finished tile rectangles.
	FinishedTiles() (map[image.Rectangle]struct{}, error)
	// GetTileImg returns the image for a specific tile rectangle.
	GetTileImg(rect image.Rectangle) (*image.RGBA, error)
	// FullImageDimensions returns the width and height of the full image.
	FullImageDimensions() (width, height int, err error)
	// TotalTilesNumber returns the total number of tiles to be rendered.
	TotalTilesNumber() (int, error)
	// Workers returns the number of workers currently running.
	Workers() (int, error)
}

// Renderer is implemented by all rendering clients (CLI and web) and called from the server.
type Renderer interface {
	// RenderTile renders a single tile of the Mandelbrot image.
	//   r: region of the Mandelbrot set to render
	//   tile: rectangle specifying the tile area
	//   imgW, imgH: full image width and height
	// Returns: image.RGBA pointer for the tile, and error if rendering fails.
	RenderTile(r Region, tile image.Rectangle, imgW, imgH int) (*image.RGBA, error)
}

// RenderTileSleepTime is used by all renderers (CLI and web) to slow down rendering, so the parallelization is more apparent.
var RenderTileSleepTime = 500 * time.Millisecond
