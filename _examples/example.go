package main

import (
	"context"
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
			fmt.Println("discovered ", device.Id)
			//if device.Id != "28:EF:D2:66:94:C2" {
			//	return
			//}
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

				th := a.GetService(hkontroller.SType_Thermostat)
				if th != nil {
					ts := th.GetCharacteristic(hkontroller.CType_CurrentTemperature)
					if ts != nil {
						val, ok := ts.Value.(float64)
						if !ok {
							return
						}
						fmt.Println("current temp: ", val)
						err = device.SubscribeToEvents(context.TODO(), a.Id, ts.Iid)
						fmt.Println("subscribe result: ", err)
					}
				}

				lb := a.GetService(hkontroller.SType_Switch)
				if lb != nil {
					on := lb.GetCharacteristic(hkontroller.CType_On)
					if on != nil {
						val, ok := on.Value.(bool)
						if !ok {
							var valFloat float64
							valFloat, ok = on.Value.(float64)
							if ok {
								val = valFloat > 0
							}
						}
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

							err = device.SubscribeToEvents(context.TODO(), a.Id, on.Iid)
							fmt.Println("subscribe result: ", err)

							time.Sleep(1 * time.Second)

							val, err = device.GetCharacteristic(a.Id, on.Iid)
							if err != nil {
								fmt.Println("error getting char value: ", err)
							}
							fmt.Println("got char value: ", val)

						}
					}
				}
			}

			time.Sleep(15 * time.Minute)
			keypair, err := hkontroller.GenerateKeyPair()
			if err != nil {
				panic(err)
			}

			err = c.PairAdd(device.Id, hkontroller.Pairing{
				Name:       "another device",
				PublicKey:  keypair.Public,
				Permission: 0,
			})
			if err != nil {
				panic(err)
			}

			pps, err := device.ListPairings()
			if err != nil {
				panic(err)
			}
			fmt.Println(pps)

			err = c.UnpairDevice(device.Id)
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

			err = c.UnpairDevice(device.Id)
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
