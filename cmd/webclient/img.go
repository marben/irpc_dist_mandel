// img.go provides simple functions to initialize and draw on an HTML canvas from Go (WASM) code.
// Intended for demo purposes: initializes a canvas and draws image tiles using browser APIs.
//
// Note: These functions are designed for use in a browser environment with WebAssembly (WASM).

package main

import (
	"image"
	"syscall/js"
)

// initCanvas initializes the HTML canvas with the given id ("myCanvas").
// Fills the canvas with the specified color.
//
// Parameters:
//
//	width, height: dimensions of the canvas in pixels
//	color: CSS color string (e.g., "#000", "red")
//
// Note: Assumes the canvas element exists in the DOM. No error is thrown if not found.
func initCanvas(width, height int, color string) {
	doc := js.Global().Get("document")
	canvas := doc.Call("getElementById", "myCanvas")
	// Assumes the canvas element exists in the DOM.
	canvas.Set("width", width)
	canvas.Set("height", height)

	ctx := canvas.Call("getContext", "2d")
	ctx.Set("fillStyle", color)
	ctx.Call("fillRect", 0, 0, width, height)
}

// drawTileToCanvas draws an *image.RGBA tile onto the canvas with id "myCanvas".
// The tile's Rect field determines its position on the canvas.
//
// Parameters:
//
//	tile: pointer to image.RGBA; tile.Rect must be set to the intended canvas region
//
// Note: Intended for use in browser/WASM context. Assumes the canvas element exists in the DOM.
func drawTileToCanvas(tile *image.RGBA) {
	doc := js.Global().Get("document")
	canvas := doc.Call("getElementById", "myCanvas")
	// Assumes the canvas element exists in the DOM.
	ctx := canvas.Call("getContext", "2d")

	// Prepare the pixel data for the browser
	jsData := js.Global().Get("Uint8ClampedArray").New(len(tile.Pix))
	js.CopyBytesToJS(jsData, tile.Pix)

	// Create the ImageData object (width/height must match the buffer)
	width := tile.Rect.Dx()
	height := tile.Rect.Dy()
	imageData := js.Global().Get("ImageData").New(jsData, width, height)

	// Draw the tile at its original coordinates on the canvas
	posX := tile.Rect.Min.X
	posY := tile.Rect.Min.Y
	ctx.Call("putImageData", imageData, posX, posY)
}
