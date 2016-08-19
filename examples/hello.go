package main

import (
	"github.com/iakud/coral"
)

func hello(val string) string {
	return "hello " + val
}

func main() {
	coral.Get("/(.*)", hello)
	coral.Run("localhost:80")
}
