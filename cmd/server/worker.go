package main

import (
	"context"
	"image"
	"image/draw"
	"log"
	"maps"
	"sync"

	api "github.com/marben/irpc_dist_mandel"
)

// imgWorkScheduler manages work on single mandelbrot image rendering
// imgWorkScheduler implements api.ImgProvider and api.TileProvider
// it uses provided api.Renderer to do rendering
type imgWorkScheduler struct {
	mRegion api.MandelRegion
	img     *image.RGBA // the "global" picture

	tilesCount   int
	workersCount int // current workers count

	ctx       context.Context
	ctxCancel context.CancelFunc

	totalPixels    int
	finishedPixels int

	unstartedTiles map[image.Rectangle]struct{}
	inProcessTiles map[image.Rectangle]struct{}
	finishedTiles  map[image.Rectangle]struct{}
	m              sync.Mutex
}

func newImgWorkScheduler(w, h int, region api.MandelRegion) *imgWorkScheduler {
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
		tilesCount:     len(allTiles),
		inProcessTiles: make(map[image.Rectangle]struct{}),
		finishedTiles:  make(map[image.Rectangle]struct{}, len(allTiles)),
		totalPixels:    w * h,
		ctx:            ctx,
		ctxCancel:      cancel,
	}
}

// FinishedTiles implements api.TileProvider
// returns rectangles of tiles that are already rendered
// (called by web client to figure out which tiles to download as image and display)
func (iws *imgWorkScheduler) FinishedTiles() (map[image.Rectangle]struct{}, error) {
	iws.m.Lock()
	defer iws.m.Unlock()

	// copy is needed to because iws.finishedTiles is mutable
	rtnMap := make(map[image.Rectangle]struct{}, len(iws.finishedTiles))
	maps.Copy(rtnMap, iws.finishedTiles)

	return rtnMap, nil
}

// FullImageDimensions implements api.TileProvider
func (iws *imgWorkScheduler) FullImageDimensions() (width int, height int, err error) {
	return iws.img.Rect.Dx(), iws.img.Rect.Dy(), nil
}

// GetTileImg implements api.TileProvider
// returns image of tileRect tile. returned image has same bounds as tileRect parameter,
// so it can be directly copied onto the full image
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

// TotalTilesCount implements [api.TileProvider].
func (iws *imgWorkScheduler) TotalTilesCount() (int, error) {
	return iws.tilesCount, nil
}

// WorkersCount implements [api.TileProvider].
func (iws *imgWorkScheduler) WorkersCount() (int, error) {
	iws.m.Lock()
	defer iws.m.Unlock()

	return iws.workersCount, nil
}

// addRenderer renders unfinished tiles using renderer
// can be called from multiple goroutines in parallel. renderers will then share the rendering
func (iws *imgWorkScheduler) addRenderer(renderer api.Renderer) error {
	iws.incActiveWorkers()
	defer iws.decActiveWorkers()

	for {
		tile, found := iws.popTile()
		if !found {
			break
		}
		tileImg, err := renderer.RenderTile(iws.mRegion, 1920, 1080, tile)
		if err != nil {
			log.Printf("render of tile %s failed: %v", tile, err)
			return nil
		}
		iws.mergeTile(tileImg)
		log.Printf("rendered: %.2f%%", iws.finished()*100)
	}
	return nil
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

// GetImage implements api.ImgProvider
// blocks until the picture is fully rendered
func (iws *imgWorkScheduler) GetImage() (*image.RGBA, error) {
	<-iws.ctx.Done() // wait for render to finish
	return iws.img, nil
}

// mergeTile draws the provided tileImg onto final image
// and marks that tile as finished
func (iws *imgWorkScheduler) mergeTile(tileImg *image.RGBA) {
	// tileImg tile contains global coordinates
	// so we use them directly to write to the big picture
	dstRect := tileImg.Bounds()

	iws.m.Lock()
	defer iws.m.Unlock()

	draw.Draw(
		iws.img,
		dstRect,              // destination rectangle
		tileImg,              // source image
		tileImg.Bounds().Min, // source start
		draw.Src,
	)

	_, found := iws.inProcessTiles[dstRect]
	if found {
		iws.finishedPixels += dstRect.Dx() * dstRect.Dy()
	}

	delete(iws.inProcessTiles, tileImg.Rect)
	iws.finishedTiles[tileImg.Rect] = struct{}{}

	if len(iws.unstartedTiles) == 0 && len(iws.inProcessTiles) == 0 {
		iws.ctxCancel()
	}
}

// finished returns fraction of finished tiles
func (iws *imgWorkScheduler) finished() float32 {
	iws.m.Lock()
	defer iws.m.Unlock()
	return float32(iws.finishedPixels) / float32(iws.totalPixels)
}

func (iws *imgWorkScheduler) incActiveWorkers() {
	iws.m.Lock()
	defer iws.m.Unlock()

	iws.workersCount++

	log.Printf("workers: %d", iws.workersCount)
}

func (iws *imgWorkScheduler) decActiveWorkers() {
	iws.m.Lock()
	defer iws.m.Unlock()

	iws.workersCount--

	log.Printf("workers: %d", iws.workersCount)
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
