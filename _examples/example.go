package main

import (
	"fmt"
	"github.com/brutella/dnssd"
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
		func(e *dnssd.BrowseEntry, pairing *hkontroller.Pairing) {
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
				ai := a.GetService(hkontroller.SType_AccessoryInfo)
				if ai == nil {
					panic("nil accessory info service")
				}
				cn := ai.GetCharacteristic(hkontroller.CType_Name)
				if cn == nil {
					panic("nil acc name")
				}
				fmt.Println("  > ", cn.Value)

				lb := a.GetService(hkontroller.SType_LightBulb)
				if lb != nil {
					on := lb.GetCharacteristic(hkontroller.CType_On)
					if on != nil {
						val, ok := on.Value.(bool)
						if ok {
							fmt.Println("   >> putting lightbulb value: ", !val)
							err := pairing.PutCharacteristic(a.Id, on.Iid, !val)
							if err != nil {
								fmt.Println("error putting char value: ", err)
							}
						}
					}
				}
			}
		},
		func(e *dnssd.BrowseEntry, pairing *hkontroller.Pairing) {
			fmt.Println("pairing disappeared")
		},
	)

	x := make(chan bool)
	<-x
}
