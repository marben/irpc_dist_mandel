# Distributed Mandelbrot Renderer Demo

This project demonstrates a distributed Mandelbrot set renderer using Go, WebAssembly (WASM), and [IRPC](https://github.com/marben/irpc). Rendering is performed entirely by clients (web and CLI), while the server coordinates and distributes work.

## Architecture Overview

- **Server**: Coordinates rendering, distributes tile work, and aggregates results. Does not perform any rendering itself.
- **Web Client (WASM)**: Runs in the browser, connects to the server via WebSocket, renders tiles, and displays progress in real time.
- **CLI Client**: Connects to the server via TCP, renders tiles, and saves the fully rendered image as a PNG file.

```
+---------+      IRPC over TCP         +---------+
|  CLI    | <------------------------> |         |
| Client  |                            |         |
+---------+                            |         |
                                       | Server  |
+---------+    IRPC over WebSocket     |         |
|  WEB    | <------------------------> |         |
| Client  |                            |         |
+---------+                            +---------+
```


## Prerequisites
- Go 1.23+
- A web browser (for the web client)

## Building and Running

### 1. Run the Server
```
cd irpc_dist_mandel/cmd/server
go run .
```

### 2. Build the Web Client (WASM)
```
cd irpc_dist_mandel/cmd/webclient
GOOS=js GOARCH=wasm go build -o ../server/static/main.wasm
# Ensure go-version dependent wasm_exec.js is present in static/
cp $(go env GOROOT)/lib/wasm/wasm_exec.js ../server/static/
# Run the Server and open http://localhost:8080 in your browser
```

### 3. Run the CLI Client
```
cd irpc_dist_mandel/cmd/cliclient
# Build and run the CLI client
go run .
# The rendered image will be saved as mandel.png
```

## How It Works
- The server listens for both TCP (CLI) and WebSocket (web) connections.
- Each client provides a renderer service; the server assigns tiles to clients for rendering.
- The web client shows progressive rendering; the CLI client requests and saves only the final image.
- All rendering is performed by clients; the server only coordinates and distributes work.

```
  CLI CLIENT                       SERVER                                       
+------------------------+        +--------------------------------------------+
| +-------------------+  |        | +--------------------+                     |
| | mandel.ImgProvider|  |        | | mandel.ImgProvider |                     |
| | (Client)          |  |        | | (Service)          |     +----------+    |
| |                   |------------>|   +GetImage()      |     | Image    |    |
| +-------------------+  |        | |                    |---->|          |    |
|                        |        | +--------------------+     |          |    |
| +-------------------+  |        |                            +----------+    |
| | mandel.Render     |  |        | +---------------+              ^    ^      |
| | (Service)         |<------------| mandel.Render |              |    |      |
| |    +RenderTile()  |  |        | | (Client)      |              |    |      |
| +-------------------+  |        | +---------------+       +--------+  |      |
+------------------------+        |   |       ^             |Render  |  |      |
                                  |   |       |             |Loop    |  |      |
                                  |   |       +-------------|        |  |      |
 WEB CLIENT                       |   |                     +--------+  |      |
+-------------------------+       |   |                                 |      |
| +--------------------+  |       |   |                                 |      |
| | mandel.Render      |  |       |   |                                 |      |
| | (Service)          |<-------------+    +-------------------------+  |      |
| |    +RenderTile()   |  |       |        | mandel.TileProvider     |  |      |
| +--------------------+  |       |        | (Service)               |--+      |
|                         |       |        |   +GetTileImg()         |         |
|                         |       |        |   +FullImageDimensions()|         |
| +--------------------+  |       |        +-------------------------+         |
| | mandel.TileProvider|  |       |                    ^                       |
| | (Client)           |  |       |                    |                       |
| |                    |-------------------------------+                       |
| +--------------------+  |       |                                            |
|                         |       |                                            |
+-------------------------+       +--------------------------------------------+
```
## Network protocol definition
- All network calls are defined in `api.go` as standard go interfaces
- To alter the function calls, change the interface definition and call `go generate api.go` which generates service and client code in `api_irpc.go`


## Project Structure
- `cmd/server/` - IRPC and http server code + static files for web client
- `cmd/webclient/` - WebAssembly client code
- `cmd/cliclient/` - CLI client code
- `render/` - Mandelbrot set rendering logic. Only used by clients.
- `api.go`, `api_irpc.go` - Shared API definitions and generated IRPC protocol code

## License
MIT

## Contributing
Pull requests and issues are welcome!

---

*This project is for demonstration purpose.*
