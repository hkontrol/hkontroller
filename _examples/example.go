package main

import (
	"bufio"
	"fmt"
	"github.com/hkontrol/hkontroller"
	"os"
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

	// load from store
	_ = c.LoadPairings()

	writeln := func(args ...interface{}) {
		fmt.Println(args...)
		fmt.Print("> ")
	}

	discoverCh, lostCh := c.StartDiscovering()
	go func() {
		for d := range discoverCh {
			writeln("discovered: ", d.Id)
		}
	}()
	go func() {
		for d := range lostCh {
			writeln("lost: ", d.Id)
		}
	}()

	var device *hkontroller.Device
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		text, _ := reader.ReadString('\n')
		// convert CRLF to LF
		text = strings.Replace(text, "\n", "", -1)

		if strings.HasPrefix(text, "help") {
			fmt.Println("commands: help")
			fmt.Println("          list")
			fmt.Println("          use <device>")
		} else if strings.HasPrefix(text, "use") {
			args := strings.Split(text, " ")
			if len(args) == 1 {
				if device == nil {
					fmt.Println("no device selected")
				} else {
					fmt.Println("selected device: ", device.Id, "\t", device.FriendlyName)
				}
				continue
			}
			if len(args) != 2 {
				fmt.Println("use <device>")
				continue
			}
			device = c.GetDevice(args[1])
			if device == nil {
				fmt.Println("device not found")
			} else {
				fmt.Println("selected device: ", device.Id, "\t", device.FriendlyName)
			}
		} else if strings.HasPrefix(text, "list") {
			for _, d := range c.GetAllDevices() {
				fmt.Println(d.Id, "\t", d.FriendlyName)
			}
		} else {
			fmt.Println("unknown command: ", text)
		}
	}
}
