package main

import (
	"context"
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
	log.Println("tcp listening on port: 8081")
	tcpListener, err := net.Listen("tcp", ":8081")
	if err != nil {
		return fmt.Errorf("net.Listen: %w", err)
	}

	imgWorkSched := newImgWorkScheduler(1920, 1080, mandel.SeahorseValley)

	// irpc service providing access to ImgProvider.GetImage as defined in api.go
	// it directs call to TiledImage.
	imgProviderService := mandel.NewImgProviderIrpcService(imgWorkSched)
	tileProviderService := mandel.NewTileProviderIrpcService(imgWorkSched)

	// each connected client is passed to the workScheduler to help with the render
	server := irpc.NewServer(irpc.WithOnConnect(func(ep *irpc.Endpoint) {
		go func() {
			log.Printf("got connection from: %s", ep.RemoteAddr())

			rc, err := mandel.NewRendererIrpcClient(ep)
			if err != nil {
				log.Printf("err: new Rendering client: %v", err)
				return
			}
			if err := imgWorkSched.addRenderer(rc); err != nil {
				log.Printf("err: render on client %q: %v", ep.RemoteAddr(), err)
				return
			}
		}()
	}))

	// we provide the GetImage call to clients, so they can save it once it is renered
	server.AddService(imgProviderService, tileProviderService)

	wsListen, httpServer := webServer(context.Background())

	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			log.Fatalf("httpServer: %v", err)
		}
	}()

	go func() {
		if err := server.Serve(tcpListener); err != nil {
			log.Fatalf("server.Serve tcp: %v", err)
		}
	}()

	go func() {
		if err := server.Serve(wsListen); err != nil {
			log.Fatalf("server.Serve ws: %v", err)
		}
	}()

	log.Printf("running mandelbrot server")
	select {}
	// return server.Serve(tcpListener)
}
