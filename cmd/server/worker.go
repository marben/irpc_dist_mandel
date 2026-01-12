package main

import (
	"context"
	"image"
	"image/draw"
	"log"
	"maps"
	"sync"

	mandel "github.com/marben/irpc_dist_mandel"
)

type imgWorkScheduler struct {
	workers int
	mRegion mandel.Region
	img     *image.RGBA

	ctx       context.Context
	ctxCancel context.CancelFunc

	totalPixels    int
	finishedPixels int

	unstartedTiles map[image.Rectangle]struct{}
	inProcessTiles map[image.Rectangle]struct{}
	finishedTiles  map[image.Rectangle]struct{}
	m              sync.Mutex
}

// FinishedTiles implements mandel.TileProvider.
func (iws *imgWorkScheduler) FinishedTiles() (map[image.Rectangle]struct{}, error) {
	iws.m.Lock()
	defer iws.m.Unlock()
	rtnMap := make(map[image.Rectangle]struct{}, len(iws.finishedTiles))
	maps.Copy(rtnMap, iws.finishedTiles)

	return rtnMap, nil
}

// FullImageDimensions implements mandel.TileProvider.
func (iws *imgWorkScheduler) FullImageDimensions() (width int, height int, err error) {
	iws.m.Lock()
	defer iws.m.Unlock()

	return iws.img.Rect.Dx(), iws.img.Rect.Dy(), nil
}

// GetTileImg implements mandel.TileProvider.
func (iws *imgWorkScheduler) GetTileImg(tileRect image.Rectangle) (*image.RGBA, error) {
	iws.m.Lock()
	defer iws.m.Unlock()

	tileImg := image.NewRGBA(tileRect)
	for y := 0; y < tileRect.Dy(); y++ {
		srcY := tileRect.Min.Y + y
		srcStart := iws.img.PixOffset(tileRect.Min.X, srcY)
		srcEnd := srcStart + tileRect.Dx()*4

		dstStart := y * tileImg.Stride
		dstEnd := dstStart + tileRect.Dx()*4

		copy(tileImg.Pix[dstStart:dstEnd], iws.img.Pix[srcStart:srcEnd])
	}

	return tileImg, nil
}

func newImgWorkScheduler(w, h int, region mandel.Region) *imgWorkScheduler {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	allTilesSlice := splitRectNoClip(img.Bounds(), 64, 64)
	allTiles := make(map[image.Rectangle]struct{}, len(allTilesSlice))
	for _, t := range allTilesSlice {
		allTiles[t] = struct{}{}
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &imgWorkScheduler{
		img:            img,
		mRegion:        region,
		unstartedTiles: allTiles,
		inProcessTiles: make(map[image.Rectangle]struct{}),
		finishedTiles:  make(map[image.Rectangle]struct{}, len(allTiles)),
		totalPixels:    w * h,
		ctx:            ctx,
		ctxCancel:      cancel,
	}
}

func (iws *imgWorkScheduler) popTile() (tile image.Rectangle, found bool) {
	iws.m.Lock()
	defer iws.m.Unlock()

	// Get unstarted tile
	if len(iws.unstartedTiles) > 0 {
		for tile = range iws.unstartedTiles {
			break
		}
		delete(iws.unstartedTiles, tile)

		// Move popped tile to currently processed tiles
		iws.inProcessTiles[tile] = struct{}{}
		return tile, true
	}

	// If there is no unstarted tile, we work again on a started one
	if len(iws.inProcessTiles) > 0 {
		for tile = range iws.inProcessTiles {
			break
		}

		return tile, true
	}

	return image.Rectangle{}, false
}

// GetImage implements mandel.Server.
func (iws *imgWorkScheduler) GetImage() (*image.RGBA, error) {
	<-iws.ctx.Done()
	return iws.img, nil
}

func (iws *imgWorkScheduler) finished() float32 {
	iws.m.Lock()
	defer iws.m.Unlock()
	return float32(iws.finishedPixels) / float32(iws.totalPixels)
}

func (iws *imgWorkScheduler) tileFinished(tileImg *image.RGBA) {
	defer log.Printf("finished: %f", iws.finished())

	rect := tileImg.Bounds()

	iws.m.Lock()
	defer iws.m.Unlock()

	draw.Draw(
		iws.img,
		tileImg.Bounds(),     // destination rectangle (global coords)
		tileImg,              // source image
		tileImg.Bounds().Min, // source start
		draw.Src,
	)

	_, found := iws.inProcessTiles[rect]
	if found {
		iws.finishedPixels += rect.Dx() * rect.Dy()
	}

	delete(iws.inProcessTiles, tileImg.Rect)
	iws.finishedTiles[tileImg.Rect] = struct{}{}

	if len(iws.unstartedTiles) == 0 && len(iws.inProcessTiles) == 0 {
		iws.ctxCancel()
	}
}

func (iws *imgWorkScheduler) incActiveWorkers() {
	iws.m.Lock()
	iws.workers++
	w := iws.workers
	iws.m.Unlock()

	log.Printf("workers: %d", w)
}

func (iws *imgWorkScheduler) decActiveWorkers() {
	iws.m.Lock()
	iws.workers--
	w := iws.workers
	iws.m.Unlock()

	log.Printf("workers: %d", w)
}

// renders unfinished tiles on provided Renderer
// can be called from multiple goroutines in parallel
func (iws *imgWorkScheduler) addRenderer(renderer mandel.Renderer) error {
	iws.incActiveWorkers()
	defer iws.decActiveWorkers()

	for {
		tile, found := iws.popTile()
		if !found {
			break
		}
		tileImg, err := renderer.RenderTile(iws.mRegion, tile, 1920, 1080)
		if err != nil {
			log.Printf("render of tile %s failed: %v", tile, err)
			return nil
		}
		// ws.img.DrawTile(tileImg)
		iws.tileFinished(tileImg)
	}
	return nil
}

// splitRectNoClip splits r into tiles of size tileW × tileH.
// Tiles at the right and bottom edges are smaller if r is not divisible.
func splitRectNoClip(r image.Rectangle, tileW, tileH int) []image.Rectangle {
	if tileW <= 0 || tileH <= 0 {
		panic("tile dimensions must be positive")
	}

	w := r.Dx()
	h := r.Dy()

	var tiles []image.Rectangle

	for oy := 0; oy < h; oy += tileH {
		th := tileH
		if oy+th > h {
			th = h - oy
		}

		for ox := 0; ox < w; ox += tileW {
			tw := tileW
			if ox+tw > w {
				tw = w - ox
			}

			tile := image.Rect(
				r.Min.X+ox,
				r.Min.Y+oy,
				r.Min.X+ox+tw,
				r.Min.Y+oy+th,
			)
			tiles = append(tiles, tile)
		}
	}

	return tiles
}
