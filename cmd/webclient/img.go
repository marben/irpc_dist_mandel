package main

import (
	"fmt"
	"image"
	"syscall/js"
	"time"
)

// displays image on the site
func displayImage(img *image.RGBA) {
	start := time.Now()
	// 1. Get the Canvas element and its 2D context
	document := js.Global().Get("document")
	canvas := document.Call("getElementById", "myCanvas")
	ctx := canvas.Call("getContext", "2d")

	width := img.Rect.Dx()
	height := img.Rect.Dy()

	// 2. Create a JS TypedArray (Uint8ClampedArray) to hold the pixel data
	// The length is width * height * 4 (RGBA)
	jsData := js.Global().Get("Uint8ClampedArray").New(len(img.Pix))

	// 3. Copy the Go byte slice into the JS TypedArray
	js.CopyBytesToJS(jsData, img.Pix)

	// 4. Create ImageData and put it on the canvas
	imageData := js.Global().Get("ImageData").New(jsData, width, height)
	ctx.Call("putImageData", imageData, 0, 0)
	elapsed := time.Since(start)
	logScreen(fmt.Sprintf("draw took %s", elapsed))
}

func initImage(width, height int, color string) {
	doc := js.Global().Get("document")
	canvas := doc.Call("getElementById", "myCanvas")

	canvas.Set("width", width)
	canvas.Set("height", height)

	ctx := canvas.Call("getContext", "2d")

	ctx.Set("fillStyle", color)
	ctx.Call("fillRect", 0, 0, width, height)
}

func drawTileToCanvas(tile *image.RGBA) {
	// 1. Get the browser context
	document := js.Global().Get("document")
	canvas := document.Call("getElementById", "myCanvas")
	ctx := canvas.Call("getContext", "2d")

	// 2. Prepare the pixel data
	// We copy the tile's Pix slice (which starts at index 0 for the tile)
	jsData := js.Global().Get("Uint8ClampedArray").New(len(tile.Pix))
	js.CopyBytesToJS(jsData, tile.Pix)

	// 3. Create the ImageData object
	// Note: ImageData always expects width/height of the buffer provided
	width := tile.Rect.Dx()
	height := tile.Rect.Dy()
	imageData := js.Global().Get("ImageData").New(jsData, width, height)

	// 4. Draw to the canvas at the tile's original coordinates
	// putImageData(data, x, y)
	posX := tile.Rect.Min.X
	posY := tile.Rect.Min.Y
	ctx.Call("putImageData", imageData, posX, posY)
}
