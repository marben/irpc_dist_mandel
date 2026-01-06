package main

import (
	"fmt"
	"log"
	"net"

	"github.com/marben/irpc"
	mandel "github.com/marben/irpc_dist_mandel"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("run: %+v", err)
	}
}

func run() error {
	// standard tcp listener
	tcpListener, err := net.Listen("tcp", ":8080")
	if err != nil {
		return fmt.Errorf("net.Listen: %w", err)
	}

	// TiledImage provides tiles to be rendered and composes them to create final image
	// also implements the ImgProvider interface defined in api.go
	img := newWorkScheduler(1920, 1080, mandel.SeahorseValley)

	// irpc service providing access to ImgProvider.GetImage as defined in api.go
	// it directs call to TiledImage.
	imgProviderService := mandel.NewImgProviderIrpcService(img)

	// each connected client is passed to the workScheduler to help with the render
	server := irpc.NewServer(irpc.WithOnConnect(func(ep *irpc.Endpoint) {
		go func() {
			log.Printf("got connection from: %s", ep.RemoteAddr())

			rc, err := mandel.NewRendererIrpcClient(ep)
			if err != nil {
				log.Printf("err: new Rendering client: %v", err)
				return
			}
			if err := img.render(rc); err != nil {
				log.Printf("err: render on client %q: %v", ep.RemoteAddr(), err)
				return
			}
		}()
	}))

	// we provide the GetImage call to clients, so they can save it once it is renered
	server.AddService(imgProviderService)

	log.Printf("running mandelbrot server")
	return server.Serve(tcpListener)
}
