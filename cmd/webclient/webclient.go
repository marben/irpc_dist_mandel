// webclient.go is a WASM web client for the distributed Mandelbrot renderer.
// It connects to the Mandelbrot server, requests tile updates, and displays rendering progress in the browser.

package main

import (
	"fmt"
	"image"
	"log"
	"syscall/js"
	"time"

	"github.com/marben/irpc"
	api "github.com/marben/irpc_dist_mandel"
	"github.com/marben/irpc_dist_mandel/render"
)

// main is the entry point for the WASM web client.
// It connects to the Mandelbrot server, sets up IRPC, and manages rendering and UI updates.
// Note: All rendering is performed by clients (web and CLI); the server only coordinates and distributes work.
func main() {
	logScreenf("Starting WASM web client...")

	// Step 1: Determine server address for WebSocket connection
	loc := js.Global().Get("window").Get("location")
	host := loc.Get("host").String()
	proto := "ws"
	if loc.Get("protocol").String() == "https:" {
		proto = "wss"
	}
	websocketUrl := proto + "://" + host + "/ws"

	// Step 2: Connect to server via WebSocket
	logScreenf("Connecting to Mandelbrot server at %s...", websocketUrl)
	websocket := js.Global().Get("WebSocket").New(websocketUrl)
	websocketRWC := NewWebsocketReadWriteCloser(websocket)
	logScreenf("WebSocket connected.")

	// Step 3: Set up IRPC endpoint and renderer service
	renderer := render.RendererImpl{OnTileRender: func(tile image.Rectangle) { logScreenf("Rendering tile: %s", tile) }}
	rendererService := api.NewRendererIrpcService(renderer)
	endpoint := irpc.NewEndpoint(websocketRWC, irpc.WithEndpointServices(rendererService))
	logScreenf("IRPC endpoint created.")

	// Step 4: Create TileProvider client for server communication
	tilesProvider, err := api.NewTileProviderIrpcClient(endpoint)
	if err != nil {
		logFatalf("Failed to create TileProvider client: %v", err)
	}
	logScreenf("TileProvider client created.")

	// Step 5: Initialize canvas with full image dimensions
	logScreenf("Requesting full image dimensions from server...")
	width, height, err := tilesProvider.FullImageDimensions()
	if err != nil {
		logFatalf("Failed to get FullImageDimensions: %v", err)
	}
	logScreenf("Dimensions: %dx%d", width, height)
	initCanvas(width, height, "#3a3a6e")
	logScreenf("Canvas initialized to dimensions %dx%d", width, height)

	// Step 6: Start tile loading loop
	logScreenf("Starting tile loading loop...")
	if err := tilesLoadLoop(tilesProvider); err != nil {
		logFatalf("tilesLoadLoop: %v", err)
	}

	// Step 7: Block main goroutine to keep WASM running
	select {}
}

// logScreenf appends a formatted message to the log element in the DOM,
func logScreenf(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)

	doc := js.Global().Get("document")
	logElem := doc.Call("getElementById", "log")
	logElem.Set("textContent", logElem.Get("textContent").String()+msg+"\n")
}

// logFatalf logs a fatal error to the log window and terminates the program.
func logFatalf(format string, a ...any) {
	logScreenf("FATAL: "+format, a...)
	log.Fatalf(format, a...)
}

// tilesLoadLoop repeatedly polls the server for finished tile rectangles and downloads them if they are not yet downloaded.
// This function drives the tile rendering progress in the web client.
//
// tp: TileProvider client for fetching tile status and images from the server.
// Returns error if any network or rendering issue occurs.
func tilesLoadLoop(tp api.TileProvider) error {
	totalTiles, err := tp.TotalTilesCount()
	if err != nil {
		return fmt.Errorf("tp.TotalTilesCount: %w", err)
	}
	hudSetTotalTiles(totalTiles)

	ourFinishedTiles := make(map[image.Rectangle]struct{})
	for {
		finishedTiles, err := tp.FinishedTiles()
		if err != nil {
			return fmt.Errorf("FinishedTiles: %w", err)
		}
		for t, _ := range finishedTiles {
			_, found := ourFinishedTiles[t]
			if !found {
				// Get tileImg from the server
				tileImg, err := tp.GetTileImg(t)
				if err != nil {
					return fmt.Errorf("get tile: %v: %w", t, err)
				}
				drawTileToCanvas(tileImg)
				ourFinishedTiles[t] = struct{}{}
			}
		}

		// Update HUD with progress
		hudSetFinishedTiles(len(finishedTiles))

		workers, err := tp.WorkersCount()
		if err != nil {
			return fmt.Errorf("tp.WorkersCount: %w", err)
		}
		hudSetWorkers(workers)

		// Sleep a bit. We are polling for brevity, instead of pushing updates from the server
		time.Sleep(250 * time.Millisecond)
	}
}

// hudSetWorkers updates the HUD to show the number of currently running workers.
// workers: number of active workers reported by the server.
func hudSetWorkers(workers int) {
	js.Global().Get("document").Call("getElementById", "workersRunning").Set("textContent", workers)
}

// hudSetFinishedTiles updates the HUD to show the number of finished tiles.
// finished: number of tiles rendered so far.
func hudSetFinishedTiles(finished int) {
	js.Global().Get("document").Call("getElementById", "tilesDone").Set("textContent", finished)
}

// hudSetTotalTiles updates the HUD to show the total number of tiles to be rendered.
// total: total number of tiles in the image.
func hudSetTotalTiles(total int) {
	js.Global().Get("document").Call("getElementById", "tilesTotal").Set("textContent", total)
}
