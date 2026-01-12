package mandel

import (
	"image"
	"time"
)

// used by all renderers to slow down rendering, so the parallelizattion is more obvious
var RenderTileSleepTime = 500 * time.Millisecond

// used by cli client to get full image once it's rendered
type ImgProvider interface {
	GetImage() (*image.RGBA, error)
}

// used by web client to show rendering progress tile by tile
type TileProvider interface {
	FinishedTiles() (map[image.Rectangle]struct{}, error)
	GetTileImg(rect image.Rectangle) (*image.RGBA, error)
	FullImageDimensions() (width, height int, err error)
}

type Renderer interface {
	RenderTile(r Region, tile image.Rectangle, imgW, imgH int) (*image.RGBA, error)
}
