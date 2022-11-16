package main

import (
	"fmt"
	"github.com/hkontrol/hkontroller"
)

func main() {
	c, err := hkontroller.NewController(
		hkontroller.NewFsStore("./.store"),
		"hkontrol",
	)
	if err != nil {
		panic(err)
	}

	// load from store
	_ = c.LoadPairings()

	discoverCh, lostCh := c.StartDiscovering()

	go func() {
		for d := range discoverCh {
			fmt.Println("discovered: ", d.Id)
		}
	}()

	go func() {
		for d := range lostCh {
			fmt.Println("lost: ", d.Id)
		}
	}()

	x := make(chan bool)
	<-x
}
