package main

import (
	"bufio"
	"fmt"
	"github.com/hkontrol/hkontroller"
	"os"
	"strconv"
	"strings"
)

func main() {
	c, err := hkontroller.NewController(
		hkontroller.NewFsStore("./.store"),
		"hkontrol",
	)
	if err != nil {
		panic(err)
	}

	devices := []*hkontroller.Device{}

	// load from store
	_ = c.LoadPairings()

	writeln := func(args ...interface{}) {
		fmt.Println(args...)
		fmt.Print("> ")
	}

	discoverCh, lostCh := c.StartDiscovery()

	verify := func(d *hkontroller.Device) {
		err := d.PairVerify()
		if err != nil {
			writeln("pair-verify err: ", err)
			return
		}
		writeln("should be connected now")
	}

	go func() {
		for d := range discoverCh {
			devices = append(devices, d)
			fmt.Println("discovered: ", d.Name)
			if d.IsPaired() {
				fmt.Println("already paired, establishing connection")
				go verify(d)
			}
			writeln()
		}
	}()
	go func() {
		for d := range lostCh {
			writeln("lost: ", d.Name)
			for i := range devices {
				if devices[i].Name == d.Name {
					devices[i] = nil
				}
			}
		}
	}()

	var device *hkontroller.Device
	reader := bufio.NewReader(os.Stdin)
	for {
		if device == nil {
			fmt.Print("> ")
		} else {
			fmt.Print(device.Name, "> ")
		}
		text, _ := reader.ReadString('\n')
		text = strings.Replace(text, "\n", "", -1)

		if strings.HasPrefix(text, "help") {
			fmt.Println("commands: help")                                       // done
			fmt.Println("          devices")                                    // done
			fmt.Println("          use <device>")                               // done
			fmt.Println("if device selected:")                                  // done
			fmt.Println("          pair <pin>")                                 // done
			fmt.Println("          unpair")                                     // done
			fmt.Println("          accessories")                                // done
			fmt.Println("          get <aid> <iid>")                            // done
			fmt.Println("          put <aid> <iid> <type:number/bool> <value>") // done
			fmt.Println("          watch <aid> <iid>")                          // done
			fmt.Println("          unwatch <aid> <iid>")                        // done
			fmt.Println("          quit")                                       // done
		} else if strings.HasPrefix(text, "use") {
			args := strings.Split(text, " ")
			if len(args) == 1 {
				if device == nil {
					fmt.Println("no device selected")
				} else {
					device = nil
				}
				continue
			}
			if len(args) != 2 {
				fmt.Println("use <device no>")
				continue
			}
			i, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				fmt.Println("error parsing num: ", err)
			}
			if i >= 0 && i < int64(len(devices)) {
				device = devices[i]
				if device == nil {
					fmt.Println("device not found")
				} else {
					fmt.Println("selected device: ", device.Name, "\t", device.Name)
				}
			}
		} else if strings.HasPrefix(text, "devices") {
			fmt.Println("#No\tID\tFriendlyName\tDNSSD\tPaired\tVerifying\tVerified")
			for i, d := range devices {
				if d == nil {
					continue
				}
				str := strconv.FormatInt(int64(i), 10) + "\t"
				str += d.Name + "\t" + d.Name
				if d.IsDiscovered() {
					str += "\tdiscovered"
				} else {
					str += "\t---"
				}
				if d.IsPaired() {
					str += "\tpaired"
				} else {
					str += "\t---"
				}
				if d.IsVerified() {
					str += "\tverified"
				} else {
					str += "\t---"
				}
				fmt.Println(str)
			}
		} else if strings.HasPrefix(text, "pair") {
			if device == nil {
				fmt.Println("no device selected")
				continue
			}
			args := strings.Split(text, " ")
			if len(args) == 1 {
				fmt.Println("enter pin")
				continue
			}
			if len(args) != 2 {
				fmt.Println("pair <pin>")
				continue
			}
			pin := args[1]
			err := device.PairSetup(pin)
			if err != nil {
				fmt.Println("pair-setup error: ", err)
				continue
			}
			fmt.Println("device paired")
			fmt.Println("establishing encrypted session")
			go verify(device)
		} else if strings.HasPrefix(text, "unpair") {
			if device == nil {
				fmt.Println("no device selected")
				continue
			}
			err := device.Unpair()
			if err != nil {
				fmt.Println("unpair err: ", err)
			}
			fmt.Println("should not be paired anymore")
		} else if strings.HasPrefix(text, "accessories") {
			if device == nil {
				fmt.Println("no device selected")
				continue
			}
			err := device.GetAccessories()
			if err != nil {
				fmt.Println("error getting accessories from device: ", err)
				continue
			}
			for _, a := range device.Accessories() {
				infoS := a.GetService(hkontroller.SType_AccessoryInfo)
				if infoS == nil {
					fmt.Println("no info service found for acc #", a.Id)
					continue
				}
				nameC := infoS.GetCharacteristic(hkontroller.CType_Name)
				if nameC == nil {
					fmt.Println("cannot find Name characteristic for info service of acc #", a.Id)
					continue
				}
				fmt.Println()
				fmt.Println("#", a.Id, nameC.Value)

				for i, s := range a.Ss {
					if i < len(a.Ss)-1 {
						fmt.Println("    │")
						fmt.Println("    ├─service: ", s.Type)
					} else {
						fmt.Println("    │")
						fmt.Println("    └─service: ", s.Type)
					}
					for j, c := range s.Cs {
						if j < len(s.Cs)-1 {
							if i == len(a.Ss)-1 {
								fmt.Println("       ├─ characteristic #", c.Iid, "\t[", c.Type, "] = ", c.Value)
							} else {
								fmt.Println("    │  ├─ characteristic #", c.Iid, "\t[", c.Type, "] = ", c.Value)
							}
						} else {
							if i == len(a.Ss)-1 {
								fmt.Println("       └─ characteristic #", c.Iid, "\t[", c.Type, "] = ", c.Value)
							} else {
								fmt.Println("    │  └─ characteristic #", c.Iid, "\t[", c.Type, "] = ", c.Value)
							}
						}
					}
				}
			}
			fmt.Println()
		} else if strings.HasPrefix(text, "get") {
			if device == nil {
				fmt.Println("no device selected")
				continue
			}
			args := strings.Split(text, " ")
			if len(args) != 3 {
				fmt.Println("get <aid> <iid>")
				continue
			}
			aid, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				fmt.Println("error parsing aid")
				continue
			}
			iid, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				fmt.Println("error parsing iid")
				continue
			}
			c, err := device.GetCharacteristic(aid, iid)
			if err != nil {
				fmt.Println("error getting characteristic value: ", err)
				continue
			}
			fmt.Println("- characteristic #", c.Iid, "\t [", c.Type, "] \t value:", c.Value)
		} else if strings.HasPrefix(text, "put") {
			if device == nil {
				fmt.Println("no device selected")
				continue
			}
			args := strings.Split(text, " ")
			if len(args) != 5 {
				fmt.Println("put <aid> <iid> <type:number/bool> <value>")
				continue
			}
			aid, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				fmt.Println("error parsing aid")
				continue
			}
			iid, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				fmt.Println("error parsing iid")
				continue
			}
			type_ := args[3]
			if type_ != "number" && type_ != "bool" {
				fmt.Println("unsupported value type")
				continue
			}
			valueStr := args[4]
			var value interface{}
			if type_ == "number" {
				valueNum, err := strconv.ParseFloat(valueStr, 64)
				if err != nil {
					fmt.Println("error parsing number: ", err)
					continue
				}
				value = valueNum
			} else {
				value = valueStr == "true" || valueStr == "1"
			}
			err = device.PutCharacteristic(aid, iid, value)
			if err != nil {
				fmt.Println("error putting characteristic")
				continue
			}
			fmt.Println("here comes success")
		} else if strings.HasPrefix(text, "watch") {
			if device == nil {
				fmt.Println("no device selected")
				continue
			}
			args := strings.Split(text, " ")
			if len(args) != 3 {
				fmt.Println("watch <aid> <iid>")
				continue
			}
			aid, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				fmt.Println("error parsing aid")
				continue
			}
			iid, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				fmt.Println("error parsing iid")
				continue
			}
			watcher, err := device.SubscribeToEvents(aid, iid)
			if err != nil {
				fmt.Println("error subscribing: ", err)
				continue
			}
			go func(d *hkontroller.Device) {
				did := d.Name
				for v := range watcher {
					fmt.Println("EVENT from ", did,
						" aid=", v.Args[0], ", iid=", v.Args[1], ", value=", v.Args[2])
					if device == nil {
						fmt.Print("> ")
					} else {
						fmt.Print(device.Name, "> ")
					}
				}
			}(device)
			fmt.Println("here comes success")
		} else if strings.HasPrefix(text, "unwatch") {
			if device == nil {
				fmt.Println("no device selected")
				continue
			}
			args := strings.Split(text, " ")
			if len(args) != 3 {
				fmt.Println("unwatch <aid> <iid>")
				continue
			}
			aid, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				fmt.Println("error parsing aid")
				continue
			}
			iid, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				fmt.Println("error parsing iid")
				continue
			}
			err = device.UnsubscribeFromEvents(aid, iid)
			if err != nil {
				fmt.Println("error unsubscribing: ", err)
				continue
			}
			fmt.Println("here comes success")
		} else if strings.HasPrefix(text, "quit") {
			for _, d := range c.GetVerifiedDevices() {
				d.Close()
			}
			os.Exit(0)
		} else {
			fmt.Println("unknown command: ", text)
		}
	}
}
