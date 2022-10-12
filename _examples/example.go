package main

import (
	"fmt"
	"github.com/brutella/dnssd"
	"github.com/hkontrol/hkontroller"
	"time"
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
		func(e *dnssd.BrowseEntry, device *hkontroller.Device) {
			if device.Id != "CC:22:3D:E3:CE:65" {
				return
			}
			err = c.PairSetup(device.Id, "031-45-154")
			if err != nil {
				panic(err)
			}
			err = c.PairVerify(device.Id)
			if err != nil {
				panic(err)
			}

			p := c.GetPairedDevice(device.Id)
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
							err := device.PutCharacteristic(a.Id, on.Iid, !val)
							if err != nil {
								fmt.Println("error putting char value: ", err)
							}

							val, err := device.GetCharacteristic(a.Id, on.Iid)
							if err != nil {
								fmt.Println("error getting char value: ", err)
							}
							fmt.Println("got char value: ", val)
						}
					}
				}
			}

			keypair, err := hkontroller.GenerateKeyPair()
			if err != nil {
				panic(err)
			}

			err = c.PairAdd(device, hkontroller.Pairing{
				Name:       "another device",
				PublicKey:  keypair.Public,
				Permission: 0,
			})
			if err != nil {
				panic(err)
			}

			_, err = device.ListPairings()
			if err != nil {
				panic(err)
			}

			time.Sleep(time.Second)
			err = c.UnpairDevice(device)
			if err != nil {
				panic(err)
			}

			// one more try
			fmt.Println("-- one more try")
			err = c.PairSetup(device.Id, "031-45-154")
			if err != nil {
				panic(err)
			}
			err = c.PairVerify(device.Id)
			if err != nil {
				panic(err)
			}

			err = c.UnpairDevice(device)
			if err != nil {
				panic(err)
			}
			fmt.Println("-- here comes success")

		},
		func(e *dnssd.BrowseEntry, d *hkontroller.Device) {
			fmt.Println("pairing disappeared")
			fmt.Println(d.DiscoverAccessories())
		},
	)

	x := make(chan bool)
	<-x
}
