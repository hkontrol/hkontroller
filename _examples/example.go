package main

import (
	"fmt"
	"github.com/brutella/dnssd"
	"github.com/hkontrol/hkontroller"
	"github.com/olebedev/emitter"
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

	fmt.Println("loaded?")

	c.StartDiscovering(
		func(e *dnssd.BrowseEntry, device *hkontroller.Device) {
			fmt.Println("discovered ", device.Id)
			if !device.IsPaired() {
				err = device.PairSetup("031-45-154")
				if err != nil {
					panic(err)
				}
			}
			if !device.IsVerified() {
				err = device.PairVerify()
				if err != nil {
					panic(err)
				}
			}

			p := c.GetDevice(device.Id)
			if p == nil {
				panic("no paired device found")
			}

			err = p.GetAccessories()
			if err != nil {
				panic(err)
			}
			fmt.Println("num of accs: ", len(p.Accessories()))
			for _, a := range p.Accessories() {
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
				fmt.Println("thermostat? ", th)
			}

			for _, a := range p.Accessories() {
				ai := a.GetService(hkontroller.SType_AccessoryInfo)
				if ai == nil {
					panic("nil accessory info service")
				}
				cn := ai.GetCharacteristic(hkontroller.CType_Name)
				if cn == nil {
					panic("nil acc name")
				}
				fmt.Println("  >>>>>> >>>> >>>  ", cn.Value)

				th := a.GetService(hkontroller.SType_Thermostat)
				if th != nil {
					ts := th.GetCharacteristic(hkontroller.CType_CurrentTemperature)
					if ts != nil {
						val, ok := ts.Value.(float64)
						if !ok {
							return
						}
						fmt.Println("current temp: ", val)
						subCh, err := device.SubscribeToEvents(a.Id, ts.Iid)
						if err == nil {
							go func() {
								for e := range subCh {
									fmt.Println("subCb: ", e.Args)
								}
								fmt.Println("subCb channel closed!")
							}()
							go func(aid, iid uint64, ch <-chan emitter.Event) {
								time.Sleep(30 * time.Second)
								fmt.Println("unsubscribing from event ", aid, iid)
								err := device.UnsubscribeFromEvents(aid, iid, ch)
								if err != nil {
									fmt.Println("unsubscribe err: ", err)
									return
								}
							}(a.Id, ts.Iid, subCh)
						}
					}
				}

				sw := a.GetService(hkontroller.SType_Switch)
				if sw != nil {
					on := sw.GetCharacteristic(hkontroller.CType_On)
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

							subCh, err := device.SubscribeToEvents(a.Id, on.Iid)
							if err == nil {
								go func() {
									for e := range subCh {
										fmt.Println("subCb: ", e.Args)
									}
									fmt.Println("subCb channel closed!")
								}()
								go func(aid, iid uint64, ch <-chan emitter.Event) {
									time.Sleep(30 * time.Second)
									fmt.Println("unsubscribing from event ", aid, iid)
									err := device.UnsubscribeFromEvents(aid, iid, ch)
									if err != nil {
										fmt.Println("unsubscribe err: ", err)
										return
									}
								}(a.Id, on.Iid, subCh)
							}

							//time.Sleep(1 * time.Second)

							val, err = device.GetCharacteristic(a.Id, on.Iid)
							if err != nil {
								fmt.Println("error getting char value: ", err)
							}
							fmt.Println("got char value: ", val)

						}
					}
				}
			}

			time.Sleep(70 * time.Second)
			keypair, err := hkontroller.GenerateKeyPair()
			if err != nil {
				panic(err)
			}

			err = device.PairAdd(hkontroller.Pairing{
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

			err = device.PairRemove("another device")
			if err != nil {
				panic(err)
			}

			pps, err = device.ListPairings()
			if err != nil {
				panic(err)
			}
			fmt.Println(pps)

			err = device.Unpair()
			if err != nil {
				panic(err)
			}

			// one more try
			fmt.Println("-- one more try")
			err = device.PairSetup("031-45-154")
			if err != nil {
				panic(err)
			}
			err = device.PairVerify()
			if err != nil {
				panic(err)
			}

			err = device.Unpair()
			if err != nil {
				panic(err)
			}
			fmt.Println("-- here comes success")

		},
		func(e *dnssd.BrowseEntry, d *hkontroller.Device) {
			fmt.Println("pairing disappeared")
			fmt.Println(d.GetAccessories())
			fmt.Println("found accessories: ", len(d.Accessories()))
		},
	)

	x := make(chan bool)
	<-x
}
