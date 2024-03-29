package hkontroller

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/hkontrol/dnssd"
	_ "github.com/hkontrol/dnssd/log"
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
	name            string
	uuid            string
	mu              sync.Mutex
	cancelDiscovery context.CancelFunc
	devices         map[string]*Device

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
		name:      name,
		mu:        sync.Mutex{},
		devices:   make(map[string]*Device),
		st:        &st,
		localLTKP: keypair.Public,
		localLTSK: keypair.Private,
	}, nil
}

func (c *Controller) putDevice(dd *Device) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.devices[dd.Name] = dd

	devPairedCh := dd.OnPaired()
	go func() {
		for range devPairedCh {
			fmt.Println("save pairing: ", dd.pairing)
			c.st.SavePairing(dd.pairing)
		}
	}()

	devUnpairedCh := dd.OnUnpaired()
	go func() {
		for range devUnpairedCh {
			c.st.DeletePairing(dd.pairing.Id)
			dd.pairing = Pairing{}
			dd.paired = false
			dd.verified = false

			c.mu.Lock()
			dd.close(errors.New("device unpaired"))
			// if not paired and not discovered,
			// then it should not present anymore
			if !dd.IsDiscovered() {
				delete(c.devices, dd.Name)
				dd.offAllTopics()
			}
			c.mu.Unlock()
		}
	}()
	devLostCh := dd.OnLost()
	go func() {
		for range devLostCh {
			if !dd.IsPaired() {
				// if lost and not paired,
				// then it should not present anymore
				c.mu.Lock()
				delete(c.devices, dd.Name)
				c.mu.Unlock()
				dd.offAllTopics()
			}
		}
	}()
}

func (c *Controller) getDevice(id string) *Device {
	c.mu.Lock()
	defer c.mu.Unlock()
	dd, ok := c.devices[id]
	if !ok {
		return nil
	}
	return dd
}

func (c *Controller) StartDiscoveryWithContext(ctx context.Context) (<-chan *Device, <-chan *Device) {

	discoverCh := make(chan *Device)
	lostCh := make(chan *Device)

	addFn := func(e dnssd.BrowseEntry) {
		name := e.Name

		dd := c.getDevice(e.Name)
		if dd == nil {
			// not exist - init one
			dd = newDevice(&e, name, c.name, c.localLTKP, c.localLTSK)
			c.putDevice(dd)
		}
		c.devices[name].mergeDnssdEntry(e)

		dd.discovered = true
		discoverCh <- dd
	}

	rmvFn := func(e dnssd.BrowseEntry) {
		//id := strings.Join([]string{e.Name, e.Type, e.Domain}, ".")
		id := e.Name
		dd := c.getDevice(id)

		if dd != nil {
			dd.discovered = false
			dd.setDnssdEntry(nil)
			dd.emit("lost")
			dd.close(errors.New("device lost from mdns"))
			lostCh <- dd
		}
	}

	go func() {
		defer func() {
			close(discoverCh)
			close(lostCh)
		}()
		for {
			newCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			c.cancelDiscovery = cancel
			if err := dnssd.LookupType(newCtx, "_hap._tcp.local.", addFn, rmvFn); err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				continue
			}
		}
	}()
	return discoverCh, lostCh
}

func (c *Controller) StartDiscovery() (<-chan *Device, <-chan *Device) {
	return c.StartDiscoveryWithContext(context.Background())
}

func (c *Controller) StopDiscovery() {
	if c.cancelDiscovery != nil {
		c.cancelDiscovery()
	}
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

	pp := c.st.Pairings()
	for _, p := range pp {
		name := p.Name
		dd := newDevice(nil, name, c.name, c.localLTKP, c.localLTSK)
		dd.pairing = p
		dd.paired = true

		c.putDevice(dd)
	}

	return nil
}

func (c *Controller) GetDevice(deviceName string) *Device {
	return c.getDevice(deviceName)
}
