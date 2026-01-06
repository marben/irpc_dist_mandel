package main

import (
	"context"
	"image"
	"image/draw"
	"log"
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

	unstarted map[image.Rectangle]struct{}
	inProcess map[image.Rectangle]struct{}
	m         sync.Mutex
}

func newWorkScheduler(w, h int, region mandel.Region) *imgWorkScheduler {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	allTilesSlice := splitRectNoClip(img.Bounds(), 64, 64)
	allTiles := make(map[image.Rectangle]struct{}, len(allTilesSlice))
	for _, t := range allTilesSlice {
		allTiles[t] = struct{}{}
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &imgWorkScheduler{
		img:         img,
		mRegion:     region,
		unstarted:   allTiles,
		inProcess:   make(map[image.Rectangle]struct{}),
		totalPixels: w * h,
		ctx:         ctx,
		ctxCancel:   cancel,
	}
}

func (iws *imgWorkScheduler) popTile() (tile image.Rectangle, found bool) {
	iws.m.Lock()
	defer iws.m.Unlock()

	// Get unstarted tile
	if len(iws.unstarted) > 0 {
		for tile = range iws.unstarted {
			break
		}
		delete(iws.unstarted, tile)

		// Move popped tile to currently processed tiles
		iws.inProcess[tile] = struct{}{}
		return tile, true
	}

	// If there is no unstarted tile, we work again on a started one
	if len(iws.inProcess) > 0 {
		for tile = range iws.inProcess {
			break
		}

		return tile, true
	}

	return image.Rectangle{}, false
}

// GetImage implements mandel.Server.
func (iws *imgWorkScheduler) GetImage() (image.RGBA, error) {
	<-iws.ctx.Done()
	return *iws.img, nil
}

func (iws *imgWorkScheduler) finished() float32 {
	iws.m.Lock()
	defer iws.m.Unlock()
	return float32(iws.finishedPixels) / float32(iws.totalPixels)
}

func (iws *imgWorkScheduler) tileFinished(tileImg image.RGBA) {
	defer log.Printf("finished: %f", iws.finished())

	rect := tileImg.Bounds()
	iws.m.Lock()
	defer iws.m.Unlock()

	draw.Draw(
		iws.img,
		tileImg.Bounds(),     // destination rectangle (global coords)
		&tileImg,             // source image
		tileImg.Bounds().Min, // source start
		draw.Src,
	)

	_, found := iws.inProcess[rect]
	if found {
		iws.finishedPixels += rect.Dx() * rect.Dy()
	}

	delete(iws.inProcess, tileImg.Rect)

	if len(iws.unstarted) == 0 && len(iws.inProcess) == 0 {
		iws.ctxCancel()
	}
}

func (iws *imgWorkScheduler) incActiveWorker() {
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
func (iws *imgWorkScheduler) render(renderer mandel.Renderer) error {
	iws.incActiveWorker()
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
