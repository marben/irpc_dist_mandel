package main

import (
	"fmt"
	"image"
	"log"
	"syscall/js"
	"time"

	"github.com/marben/irpc"
	mandel "github.com/marben/irpc_dist_mandel"
	"github.com/marben/irpc_dist_mandel/render"
)

func logScreen(msg string) {
	doc := js.Global().Get("document")
	log := doc.Call("getElementById", "log")
	log.Set("textContent", log.Get("textContent").String()+msg+"\n")
}

func tilesLoadLoop(tp mandel.TileProvider) error {
	ourFinishedTiles := make(map[image.Rectangle]struct{})
	for {
		finTiles, err := tp.FinishedTiles()
		if err != nil {
			return fmt.Errorf("FinishedTiles: %w", err)
		}
		for t, _ := range finTiles {
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

		// Polling for brevity
		time.Sleep(250 * time.Millisecond)
	}
}

func main() {
	log.Println("main is running.")
	logScreen("main is running")

	// Figure out the server address to open WebSocket
	loc := js.Global().Get("window").Get("location")
	host := loc.Get("host").String()
	proto := "ws"
	if loc.Get("protocol").String() == "https:" {
		proto = "wss"
	}

	wsUrl := proto + "://" + host + "/ws"

	// Connect WebSocket
	// ws := js.Global().Get("WebSocket").New("ws://localhost:8080/ws")
	ws := js.Global().Get("WebSocket").New(wsUrl)
	wsRWC := NewWSReadWriteCloser(ws)

	// Establish IRPC connection (Endpoint) over Websocket
	rendererService := mandel.NewRendererIrpcService(render.RendererImpl{})
	ep := irpc.NewEndpoint(wsRWC, irpc.WithEndpointServices(rendererService))
	logScreen("ep created")

	tilesProvider, err := mandel.NewTileProviderIrpcClient(ep)
	if err != nil {
		log.Fatalf("NewTileProviderIrpcClient: %v", err)
	}

	width, height, err := tilesProvider.FullImageDimensions()
	if err != nil {
		log.Fatalf("getting FullImageDimensions(): %w", err)
	}

	initImage(width, height, "#3a3a6e")
	if err := tilesLoadLoop(tilesProvider); err != nil {
		logScreen(fmt.Sprintf("tilesLoadLoop: %v", err))
		log.Fatalf("tilesLoadLoop: %v", err)
	}

	// Prevent Go program from exiting
	select {}
}
