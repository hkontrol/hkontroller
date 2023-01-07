package hkontroller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/brutella/dnssd"
	_ "github.com/brutella/dnssd/log"
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

func (c *Controller) StartDiscovering() (<-chan *Device, <-chan *Device) {

	discoverCh := make(chan *Device)
	lostCh := make(chan *Device)

	addFn := func(e dnssd.BrowseEntry) {
		id := strings.Join([]string{e.Name, e.Type, e.Domain}, ".")
		c.mu.Lock()
		c.mdnsDiscovered[id] = &e

		dd, ok := c.devices[id]
		if !ok {
			// not exist - init one
			c.devices[id] = newDevice(&e, id, c.name, c.localLTKP, c.localLTSK)
			pairing := Pairing{Name: id}
			c.devices[id].pairing = pairing
			dd = c.devices[id]
			devPairedCh := dd.OnPaired()
			go func() {
				for range devPairedCh {
					c.st.SavePairing(dd.pairing)
				}
			}()

			devUnpairedCh := dd.OnUnpaired()
			go func() {
				for range devUnpairedCh {
					// if not paired and not discovered, then it should not present anymore
					// TODO: create separate method
					c.mu.Lock()
					dd.close()
					if !dd.IsDiscovered() {
						delete(c.devices, dd.Id)
						dd.offAllTopics()
					}
					c.mu.Unlock()
					c.st.DeletePairing(dd.Id)
				}
			}()
		}
		c.devices[id].mergeDnssdEntry(e)

		dd.discovered = true
		c.mu.Unlock()
		// end section tcp conn
		discoverCh <- dd
	}

	rmvFn := func(e dnssd.BrowseEntry) {
		id := strings.Join([]string{e.Name, e.Type, e.Domain}, ".")
		c.mu.Lock()
		delete(c.mdnsDiscovered, id)
		dd, ok := c.devices[id]
		c.mu.Unlock()

		if ok {
			dd.discovered = false
			dd.setDnssdEntry(nil)
			dd.emit("lost")
			dd.close()
			lostCh <- dd
			if !dd.IsPaired() {
				// if not paired and not discovered, then it should not present anymore
				c.mu.Lock()
				delete(c.devices, dd.Id)
				c.mu.Unlock()
				dd.offAllTopics()
			}
		}
	}

	go func() {
		defer func() {
			close(discoverCh)
			close(lostCh)
		}()
		for {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			c.cancelDiscovering = cancel
			if err := dnssd.LookupType(ctx, "_hap._tcp.local.", addFn, rmvFn); err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				continue
			}
		}
	}()
	return discoverCh, lostCh
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

// GetAllDevices returns list of all devices loaded or discovered by controller.
func (c *Controller) GetAllDevices() []*Device {
	var result []*Device

	c.mu.Lock()
	defer c.mu.Unlock()
	for _, d := range c.devices {
		result = append(result, d)
	}

	return result
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
		c.devices[id] = newDevice(nil, id, c.name, c.localLTKP, c.localLTSK)
		c.devices[id].pairing = p
		c.devices[id].paired = true

		dd := c.devices[id]

		devPairedCh := dd.OnPaired()
		go func() {
			for range devPairedCh {
				c.st.SavePairing(dd.pairing)
			}
		}()

		devUnpairedCh := dd.OnUnpaired()
		go func() {
			for range devUnpairedCh {
				// if not paired and not discovered, then it should not present anymore
				c.mu.Lock()
				dd.close()
				if !dd.IsDiscovered() {
					delete(c.devices, dd.Id)
					dd.offAllTopics()
				}
				c.mu.Unlock()
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
