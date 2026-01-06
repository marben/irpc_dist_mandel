package mandel

import (
	"image"
)

type ImgProvider interface {
	GetImage() (image.RGBA, error)
}

type Renderer interface {
	RenderTile(r Region, tile image.Rectangle, imgW, imgH int) (image.RGBA, error)
}
