package main

import "github.com/kfcemployee/goserver/internal"

func main() {
	internal.Test([4]byte{127, 0, 0, 1}, 8080)
}
