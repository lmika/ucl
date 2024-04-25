package main

import (
	"context"
)

func main() {
	initJS(context.Background())

	<-make(chan struct{})
}
