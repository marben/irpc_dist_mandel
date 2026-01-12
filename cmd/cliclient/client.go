package main

import (
	"fmt"
	"image/png"
	"log"
	"net"
	"os"

	"github.com/marben/irpc"
	mandel "github.com/marben/irpc_dist_mandel"
	"github.com/marben/irpc_dist_mandel/render"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("run: %+v", err)
	}
}

func run() error {
	log.Printf("connecting")
	tcpConn, err := net.Dial("tcp", ":8081")
	if err != nil {
		return err
	}

	// create the renderer service, that will be called from server to render tiles
	rendererService := mandel.NewRendererIrpcService(render.RendererImpl{})
	ep := irpc.NewEndpoint(tcpConn, irpc.WithEndpointServices(rendererService))

	client, err := mandel.NewImgProviderIrpcClient(ep)
	if err != nil {
		return err
	}

	log.Println("asking server for currently rendered image")
	img, err := client.GetImage()
	if err != nil {
		return fmt.Errorf("client.GetImage: %w", err)
	}

	// Save rendered file
	filename := "mandel.png"
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		return err
	}

	log.Printf("fully rendered file saved to %q", filename)

	return nil
}
