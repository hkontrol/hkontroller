package hkontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/brutella/dnssd"
	"log"
	"net"
	"net/http"
	"sort"
	"sync"
	"time"
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
	pairings          map[string]*Pairing

	st *storer

	localLTKP []byte
	localLTSK []byte
}

func NewController(store Store, name string) (*Controller, error) {

	st := storer{store}

	kpair, err := st.KeyPair()
	if err != nil {
		keypair, err := generateKeyPair()
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
		pairings:       make(map[string]*Pairing),
		st:             &st,
		localLTKP:      kpair.Public,
		localLTSK:      kpair.Private,
	}, nil
}

func (c *Controller) StartDiscovering(onDiscover func(*dnssd.BrowseEntry, *Pairing), onRemove func(*dnssd.BrowseEntry, *Pairing)) {
	addFn := func(e dnssd.BrowseEntry) {
		// CC:22:3D:E3:CE:65 example of id
		id, ok := e.Text["id"]
		if !ok {
			return
		}
		c.mu.Lock()
		c.mdnsDiscovered[id] = &e

		pairing, ok := c.pairings[id]
		if !ok {
			// not exist - init one
			c.pairings[id] = &Pairing{
				Name: id,
			}
			pairing = c.pairings[id]
		}
		c.mu.Unlock()

		pairing.discovered = true

		if len(e.IPs) == 0 {
			c.mu.Unlock()
			return
		}

		sort.Slice(e.IPs, func(i, j int) bool {
			return e.IPs[i].To4() == nil // currently ip6 first to check following
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
			log.Println("dialing ", devTcpAddr)
			dial, err := net.DialTimeout("tcp", devTcpAddr, 10*time.Millisecond)
			if err != nil {
				log.Println("tcpAddr: ", devTcpAddr, "error : ", err)
				continue
			}
			// connection ok, close it and break the loop
			dial.Close()
			break
		}

		// tcp conn open
		dial, err := net.Dial("tcp", devTcpAddr)
		if err != nil {
			return
		}
		// connection, http client
		cc := newConn(dial)

		pairing.httpc = &http.Client{
			Transport: cc,
		}
		pairing.cc = cc
		pairing.tcpAddr = devTcpAddr
		pairing.httpAddr = devHttpUrl

		// end section tcp conn
		onDiscover(&e, pairing)
	}

	rmvFn := func(e dnssd.BrowseEntry) {
		id, ok := e.Text["id"]
		if !ok {
			return
		}
		c.mu.Lock()
		delete(c.mdnsDiscovered, id)
		pairing, ok := c.pairings[id]
		c.mu.Unlock()

		if ok {
			pairing.discovered = false
			pairing.verified = false
			pairing.httpc = nil
			onRemove(&e, pairing)
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

	for k, v := range c.pairings {
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

func (c *Controller) LoadPairings() error {

	c.mu.Lock()
	defer c.mu.Unlock()

	pp := c.st.Pairings()
	for _, p := range pp {
		p.paired = true
		c.pairings[p.Name] = &p
	}

	return nil
}

func (c *Controller) GetPairing(deviceId string) *Pairing {
	c.mu.Lock()
	defer c.mu.Unlock()

	a, ok := c.pairings[deviceId]
	if !ok {
		return nil
	}
	return a
}
