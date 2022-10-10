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

	c.StartDiscovering(
		func(pairing *hkontroller.Pairing) {
			if pairing.Name != "CC:22:3D:E3:CE:65" {
				return
			}
			err = c.PairSetup(pairing.Name, "031-45-154")
			if err != nil {
				panic(err)
			}
			err = c.PairVerify(pairing.Name)
			if err != nil {
				panic(err)
			}

			p := c.GetPairing(pairing.Name)
			if p == nil {
				panic("no paired device found")
			}

			err = p.DiscoverAccessories()
			if err != nil {
				panic(err)
			}
			fmt.Println("num of accs: ", len(p.GetAccessories()))

			for _, a := range p.GetAccessories() {
				ai := a.GetAccessoryInfoService()
				if ai == nil {
					panic("nil accessory info service")
				}
				for _, c := range ai.Cs {
					if *c.Type == hkontroller.Name {
						fmt.Println("   > ", c.Value)
					}
				}
			}
		},
		func(pairing *hkontroller.Pairing) {
			fmt.Println("pairing disappeared")
		},
	)

	x := make(chan bool)
	<-x
}
