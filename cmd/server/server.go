package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/marben/irpc"
	mandel "github.com/marben/irpc_dist_mandel"
)

// main is the entry point for the Mandelbrot server.
// Note: All rendering is performed by clients (web and CLI); the server only coordinates and distributes work.
func main() {
	if err := run(); err != nil {
		log.Fatalf("run: %+v", err)
	}
}

func run() error {
	// replace mandel.SeahorseValley with other predefined region to see other parts of mb set
	imgWorkScheduler := newImgWorkScheduler(1920, 1080, mandel.SeahorseValley)

	// imgProviderIrpcService provides mandel.ImgProvider interface over network
	// It defines only one function GetImage(), which returns the full image upon complete render
	// It is used by our cli clients.
	// ImgProvider is implemented by imgWorkScheduler, so we use our one instance to back the service
	imgProviderIrpcService := mandel.NewImgProviderIrpcService(imgWorkScheduler)

	// tileProviderIrpcService provides mandel.TileProvider interface over network
	// It provides many different functions to provide web clients a view of progressive rendering, workers number etc
	// TileProvider is also iplemented by imgWorkScheduler, so we use the same instance as with imgProvderIrpcSevice
	// to share computational power among both cli and web clients
	tileProviderIrpcService := mandel.NewTileProviderIrpcService(imgWorkScheduler)

	// irpc server with onConnect hook to plug clients into rendering
	irpcServer := irpc.NewServer(irpc.WithOnConnect(func(ep *irpc.Endpoint) {
		go func() {
			log.Printf("got connection from: %s", ep.RemoteAddr())

			// Each client needs to provide us with mandel.Renderer so we can use it to render tiles of full image
			rendererIrpcClient, err := mandel.NewRendererIrpcClient(ep)
			if err != nil {
				log.Printf("err: new Rendering client: %v", err)
				return
			}

			// Each connected client is used as a worker
			if err := imgWorkScheduler.addRenderer(rendererIrpcClient); err != nil {
				log.Printf("err: render on client %q: %v", ep.RemoteAddr(), err)
				return
			}
		}()
	}))

	// irpc services need to be registered to server so clients can use them
	irpcServer.AddService(imgProviderIrpcService, tileProviderIrpcService)

	// TCP
	log.Println("tcp listening on port: 8081")
	tcpListener, err := net.Listen("tcp", ":8081")
	if err != nil {
		return fmt.Errorf("net.Listen: %w", err)
	}

	// WEBSOCKET
	websocketListener, httpServer := webServer(context.Background(), 8080)

	// httpServer provides index.html, main.wasm along with websocket endpoint
	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			log.Fatalf("httpServer: %v", err)
		}
	}()

	// irpcServer can serve multiple multiple listeners. In this case both tcp and websocket
	go func() {
		if err := irpcServer.Serve(tcpListener); err != nil {
			log.Fatalf("server.Serve tcp: %v", err)
		}
	}()
	go func() {
		if err := irpcServer.Serve(websocketListener); err != nil {
			log.Fatalf("server.Serve ws: %v", err)
		}
	}()

	log.Printf("mb server waiting for tcp and websocket connections")
	select {}
}
