module http_basic

go 1.23

toolchain go1.24.4

require github.com/gurizzu/go-reqws v0.0.0

require github.com/coder/websocket v1.8.14 // indirect

replace github.com/gurizzu/go-reqws => ../..
