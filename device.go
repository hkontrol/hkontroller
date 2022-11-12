package hkontroller

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type aidIid struct {
	aid uint64
	iid uint64
}

type eventCallback func(aid uint64, iid uint64, value interface{})

type Device struct {
	Id string

	controllerId   string
	controllerLTPK []byte
	controllerLTSK []byte

	pairing Pairing

	discovered bool // discovered via mdns?
	paired     bool // completed /pair-setup?
	verified   bool // is connection established after /pair-verify?

	tcpAddr  string // tcp socket address
	httpAddr string // tcp socket address

	cc    *conn
	ss    *session
	httpc *http.Client // http client with encryption support
	accs  *Accessories

	emu           sync.Mutex
	eventHandlers map[aidIid]eventCallback
}

func newDevice(id string, controllerId string, controllerLTPK []byte, controllerLTSK []byte) *Device {
	return &Device{
		Id:             id,
		controllerId:   controllerId,
		controllerLTPK: controllerLTPK,
		controllerLTSK: controllerLTSK,
		eventHandlers:  make(map[aidIid]eventCallback),
	}
}

func (d *Device) doRequest(req *http.Request) (*http.Response, error) {
	if d.httpc == nil || d.cc == nil {
		return nil, errors.New("no http client available")
	}
	return d.httpc.Do(req)
}

func (d *Device) doPost(url string, contentType string, body io.Reader) (*http.Response, error) {
	if d.httpc == nil || d.cc == nil {
		return nil, errors.New("no http client available")
	}
	return d.httpc.Post(url, contentType, body)
}
func (d *Device) doGet(url string) (*http.Response, error) {
	if d.httpc == nil || d.cc == nil {
		return nil, errors.New("no http client available")
	}
	return d.httpc.Get(url)
}

func (d *Device) connect() error {

	if d.cc != nil {
		fmt.Println("device.connect closing old connection ")
		d.cc.Close()
		d.cc = nil
		d.httpc = nil
		fmt.Println("device.connect old connection closed")
	}
	d.verified = false

	fmt.Println("device.connect dialing ", d.tcpAddr)
	dial, err := net.DialTimeout("tcp", d.tcpAddr, time.Second)
	if err != nil {
		fmt.Println("device.connect: ", err)
		return err
	}
	fmt.Println("device.connect dial success")

	// connection, http client
	cc := newConn(dial)
	d.cc = cc
	d.httpc = &http.Client{
		Transport: d,
	}
	d.cc.SetEventCallback(d.onEvent)

	fmt.Println("device.connect returning from func")

	return nil
}

func (d *Device) startBackgroundRead() {
	go func() {
		for {
			d.cc.loop()
			if d.cc.closed {
				for {
					err := d.connect()
					if err == nil {
						break
					}
					time.Sleep(time.Second)
				}
				go func() {
					err := d.PairVerify()
					if err != nil {
						return
					}
				}()
				return
			}
		}
	}()
	d.cc.inBackground = true
}

// IsDiscovered indicates if device is advertised via multicast dns
func (d *Device) IsDiscovered() bool {
	return d.discovered
}

// IsPaired returns true if device is paired by this controller.
// If another client is paired with device it will return false.
func (d *Device) IsPaired() bool {
	return d.paired
}

// IsVerified returns true if /pair-verify step was completed by this controller.
func (d *Device) IsVerified() bool {
	return d.verified
}

func (d *Device) DiscoverAccessories() error {

	if !d.verified || d.httpc == nil {
		return errors.New("paired device not verified or not connected")
	}

	res, err := d.doGet("/accessories")
	if err != nil {
		return err
	}
	all, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	var accs Accessories
	err = json.Unmarshal(all, &accs)
	if err != nil {
		return err
	}

	// now sort to prepare for binary search
	sort.Slice(accs.Accs, func(i, j int) bool {
		return accs.Accs[i].Id < accs.Accs[j].Id
	})
	for _, a := range accs.Accs {
		// sort by service type
		sort.Slice(a.Ss, func(i, j int) bool {
			return strings.Compare(string(a.Ss[i].Type), string(a.Ss[j].Type)) < 0
		})
		for _, s := range a.Ss {
			// sort by characteristic type
			sort.Slice(s.Cs, func(i, j int) bool {
				return strings.Compare(string(s.Cs[i].Type), string(s.Cs[j].Type)) < 0
			})
		}
	}

	d.accs = &accs

	return nil
}

func (d *Device) GetAccessories() []*Accessory {
	return d.accs.Accs
}

func (d *Device) GetCharacteristic(aid uint64, cid uint64) (CharacteristicDescription, error) {
	ep := fmt.Sprintf("/characteristics?id=%d.%d", aid, cid)
	res, err := d.doGet(ep)
	if err != nil {
		return CharacteristicDescription{}, err
	}

	all, err := io.ReadAll(res.Body)
	if err != nil {
		return CharacteristicDescription{}, err
	}

	type responsePayload struct {
		Characteristics []CharacteristicDescription `json:"characteristics"`
	}

	var chrs responsePayload
	err = json.Unmarshal(all, &chrs)
	if err != nil {
		return CharacteristicDescription{}, err
	}

	for _, c := range chrs.Characteristics {
		if c.Aid == aid || c.Iid == cid {
			return c, nil
		}
	}

	return CharacteristicDescription{}, errors.New("wrong response")
}

func (d *Device) PutCharacteristic(aid uint64, cid uint64, val interface{}) error {

	type putPayload struct {
		Cs []CharacteristicPut `json:"characteristics"`
	}

	c := putPayload{Cs: []CharacteristicPut{{Aid: aid, Iid: cid, Value: val}}}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", "/characteristics", bytes.NewReader(b))
	if err != nil {
		return err
	}

	_, err = d.doRequest(req)
	if err != nil {
		return err
	}

	return nil
}

func (d *Device) onEvent(res *http.Response) {
	all, err := io.ReadAll(res.Body)
	if err != nil {
		return
	}

	type responsePayload struct {
		Characteristics []CharacteristicDescription `json:"characteristics"`
	}

	var chrs responsePayload
	err = json.Unmarshal(all, &chrs)
	if err != nil {
		return
	}
	for _, ch := range chrs.Characteristics {
		aid := ch.Aid
		iid := ch.Iid
		val := ch.Value

		ai := aidIid{
			aid: aid,
			iid: iid,
		}
		d.emu.Lock()
		cb, ok := d.eventHandlers[ai]
		if ok {
			if cb != nil {
				cb(aid, iid, val)
			}
		}
		d.emu.Unlock()
	}
}

func (d *Device) UnsubscribeFromEvents(aid uint64, cid uint64) error {
	ai := aidIid{
		aid: aid,
		iid: cid,
	}
	d.emu.Lock()
	_, ok := d.eventHandlers[ai]
	d.emu.Unlock()
	if !ok {
		return errors.New("not subscribed")
	}

	type putPayload struct {
		Cs []CharacteristicPut `json:"characteristics"`
	}

	ev := false
	c := putPayload{Cs: []CharacteristicPut{{Aid: aid, Iid: cid, Events: &ev}}}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", "/characteristics", bytes.NewReader(b))
	if err != nil {
		return err
	}

	_, err = d.doRequest(req)
	if err != nil {
		return err
	}

	d.emu.Lock()
	delete(d.eventHandlers, ai)
	d.emu.Unlock()

	return nil
}
func (d *Device) SubscribeToEvents(aid uint64, cid uint64, callback eventCallback) error {

	ai := aidIid{
		aid: aid,
		iid: cid,
	}
	d.emu.Lock()
	_, ok := d.eventHandlers[ai]
	d.emu.Unlock()
	if ok {
		return errors.New("already subscribed")
	}

	type putPayload struct {
		Cs []CharacteristicPut `json:"characteristics"`
	}

	ev := true
	c := putPayload{Cs: []CharacteristicPut{{Aid: aid, Iid: cid, Events: &ev}}}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", "/characteristics", bytes.NewReader(b))
	if err != nil {
		return err
	}

	res, err := d.doRequest(req)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusNoContent {
		return errors.New("not 204")
	}

	d.emu.Lock()
	d.eventHandlers[ai] = callback
	d.emu.Unlock()

	return nil
}
