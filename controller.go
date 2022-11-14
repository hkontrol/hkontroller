package hkontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/brutella/dnssd"
)

type pairSetupPayload struct {
	Method        byte   `tlv8:"0"`
	Identifier    string `tlv8:"1"`
	Salt          []byte `tlv8:"2"`
	PublicKey     []byte `tlv8:"3"`
	Proof         []byte `tlv8:"4"`
	EncryptedData []byte `tlv8:"5"`
	State         byte   `tlv8:"6"`
	Error         byte   `tlv8:"7"`
	RetryDelay    byte   `tlv8:"8"`
	Certificate   []byte `tlv8:"9"`
	Signature     []byte `tlv8:"10"`
	Permissions   byte   `tlv8:"11"`
	FragmentData  []byte `tlv8:"13"`
	FragmentLast  []byte `tlv8:"14"`
}

type Controller struct {
	name              string
	uuid              string
	mu                sync.Mutex
	cancelDiscovering context.CancelFunc
	mdnsDiscovered    map[string]*dnssd.BrowseEntry
	devices           map[string]*Device

	st *storer

	localLTKP []byte
	localLTSK []byte
}

func NewController(store Store, name string) (*Controller, error) {

	st := storer{store}

	keypair, err := st.KeyPair()
	if err != nil {
		keypair, err = generateKeyPair()
		if err != nil {
			return nil, fmt.Errorf("generating keypair failed: %v", err)
		}
		if err := st.SaveKeyPair(keypair); err != nil {
			return nil, fmt.Errorf("saving keypair failed: %v", err)
		}
	}

	return &Controller{
		name:           name,
		mu:             sync.Mutex{},
		mdnsDiscovered: make(map[string]*dnssd.BrowseEntry),
		devices:        make(map[string]*Device),
		st:             &st,
		localLTKP:      keypair.Public,
		localLTSK:      keypair.Private,
	}, nil
}

func (c *Controller) StartDiscovering(onDiscover func(*dnssd.BrowseEntry, *Device), onRemove func(*dnssd.BrowseEntry, *Device)) {
	addFn := func(e dnssd.BrowseEntry) {
		// CC:22:3D:E3:CE:65 example of id
		id, ok := e.Text["id"]
		if !ok {
			return
		}
		c.mu.Lock()
		c.mdnsDiscovered[id] = &e

		dd, ok := c.devices[id]
		if !ok {
			// not exist - init one
			c.devices[id] = newDevice(id, c.name, c.localLTKP, c.localLTSK)
			pairing := Pairing{Name: id}
			c.devices[id].pairing = pairing
			dd = c.devices[id]
			devPairedCh := dd.ee.On("paired")
			go func() {
				for _ = range devPairedCh {
					c.st.SavePairing(dd.pairing)
				}
			}()

			devUnpairedCh := dd.ee.On("unpaired")
			go func() {
				for _ = range devUnpairedCh {
					c.st.DeletePairing(dd.Id)
				}
			}()
		}

		c.mu.Unlock()

		dd.discovered = true
		if len(e.IPs) == 0 {
			return
		}

		sort.Slice(e.IPs, func(i, j int) bool {
			return e.IPs[i].To4() != nil // ip4 first
		})

		var devTcpAddr string
		var devHttpUrl string
		// probe every ip tcpAddr
		for _, ip := range e.IPs {
			if ip.To4() == nil {
				// ipv6 tcpAddr in square brackets
				// [fe80::...%wlp2s0]:51510
				devTcpAddr = fmt.Sprintf("[%s%%%s]:%d", ip.String(), e.IfaceName, e.Port)
				devHttpUrl = fmt.Sprintf("http://[%s]:%d", ip.String(), e.Port)
			} else {
				devTcpAddr = fmt.Sprintf("%s:%d", ip.String(), e.Port)
				devHttpUrl = fmt.Sprintf("http://%s", devTcpAddr)
			}
			dial, err := net.DialTimeout("tcp", devTcpAddr, 1000*time.Millisecond)
			if err != nil {
				continue
			}
			// connection ok, close it and break the loop
			dial.Close()
			break
		}

		dd.tcpAddr = devTcpAddr
		dd.httpAddr = devHttpUrl

		// end section tcp conn
		onDiscover(&e, dd)
	}

	rmvFn := func(e dnssd.BrowseEntry) {
		id, ok := e.Text["id"]
		if !ok {
			return
		}
		c.mu.Lock()
		delete(c.mdnsDiscovered, id)
		dd, ok := c.devices[id]
		c.mu.Unlock()

		if ok {
			dd.close()
			dd.discovered = false
			dd.verified = false
			dd.httpc = nil
			onRemove(&e, dd)
		}
	}

	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		c.cancelDiscovering = cancel
		if err := dnssd.LookupType(ctx, "_hap._tcp.local.", addFn, rmvFn); err != nil {
			return
		}
	}()
}

func (c *Controller) SavePairings(s Store) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for k, v := range c.devices {
		key := fmt.Sprintf("pairing_%s", k)

		val, err := json.Marshal(v)
		if err != nil {
			return err
		}

		err = s.Set(key, val)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetPairedDevices returns list of devices that has been paired.
// Connected or not.
func (c *Controller) GetPairedDevices() []*Device {
	var result []*Device

	c.mu.Lock()
	defer c.mu.Unlock()
	for _, d := range c.devices {
		if d.paired {
			result = append(result, d)
		}
	}

	return result
}

// GetVerifiedDevices returns list of devices with established encrypted session.
func (c *Controller) GetVerifiedDevices() []*Device {
	var result []*Device

	c.mu.Lock()
	defer c.mu.Unlock()
	for _, d := range c.devices {
		if d.verified {
			result = append(result, d)
		}
	}

	return result
}

func (c *Controller) LoadPairings() error {

	c.mu.Lock()
	defer c.mu.Unlock()

	pp := c.st.Pairings()
	for _, p := range pp {
		id := p.Name
		c.devices[id] = newDevice(id, c.name, c.localLTKP, c.localLTSK)
		c.devices[id].pairing = p
		c.devices[id].paired = true

		dd := c.devices[id]

		devPairedCh := dd.ee.On("paired")
		go func() {
			for _ = range devPairedCh {
				c.st.SavePairing(dd.pairing)
			}
		}()

		devUnpairedCh := dd.ee.On("unpaired")
		go func() {
			for _ = range devUnpairedCh {
				c.st.DeletePairing(dd.Id)
			}
		}()
	}

	return nil
}

func (c *Controller) GetDevice(deviceId string) *Device {
	c.mu.Lock()
	defer c.mu.Unlock()

	a, ok := c.devices[deviceId]
	if !ok {
		return nil
	}
	return a
}
