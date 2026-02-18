// cliclient.go is a CLI client for the distributed Mandelbrot renderer.
// It connects to the Mandelbrot server, requests the fully rendered image, and saves it as a PNG file.

package main

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"net"
	"os"

	"github.com/marben/irpc"
	api "github.com/marben/irpc_dist_mandel"
	"github.com/marben/irpc_dist_mandel/render"
)

// main is the entry point for the CLI client.
// It runs the client logic and logs any fatal errors.
// Note: All rendering is performed by clients (web and CLI); the server only coordinates and distributes work.
func main() {
	log.Printf("Starting CLI client...")
	if err := run(); err != nil {
		log.Fatalf("FATAL: %v", err)
	}
}

// run connects to the Mandelbrot server, requests the rendered image, and saves it as a PNG file.
// Returns an error if any step fails.
func run() error {
	// Step 1: Connect to Mandelbrot server
	log.Printf("Connecting to Mandelbrot server on :8081...")
	tcpConn, err := net.Dial("tcp", ":8081")
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	// Step 2: Create the renderer service, which the server can call to render tiles using our CPU
	renderer := render.RendererImpl{OnTileRender: func(tile image.Rectangle) { log.Printf("Rendering tile: %s", tile) }}
	rendererService := api.NewRendererIrpcService(renderer)
	ep := irpc.NewEndpoint(tcpConn, irpc.WithEndpointServices(rendererService))

	// Step 3: Create a client for the ImgProvider interface
	log.Printf("Creating ImgProvider client...")
	client, err := api.NewImgProviderIrpcClient(ep)
	if err != nil {
		return fmt.Errorf("failed to create ImgProvider client: %w", err)
	}

	// Step 4: Request the fully rendered image from the server
	log.Printf("Requesting fully rendered image from server...")
	img, err := client.GetImage()
	if err != nil {
		return fmt.Errorf("client.GetImage: %w", err)
	}

	// Step 5: Save the rendered image to a PNG file
	filename := "mandel.png"
	log.Printf("Saving rendered image to %q...", filename)
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		return fmt.Errorf("failed to encode PNG: %w", err)
	}

	log.Printf("Fully rendered image saved to %q", filename)
	return nil
}
