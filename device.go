package hkontroller

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/olebedev/emitter"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"
)

type Device struct {
	ee emitter.Emitter

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
}

type roundTripper struct {
	d *Device
}

func newRoundTripper(d *Device) *roundTripper {
	return &roundTripper{d: d}
}

// RoundTrip implementation to be able to use with http.Client
func (r *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {

	err := req.Write(r.d.cc)
	if err != nil {
		return nil, err
	}

	if r.d.cc.inBackground {
		res := <-r.d.cc.response
		return res, nil
	}

	rd := bufio.NewReader(r.d.cc)
	res, err := http.ReadResponse(rd, nil)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func newDevice(id string, controllerId string, controllerLTPK []byte, controllerLTSK []byte) *Device {
	d := &Device{
		Id:             id,
		controllerId:   controllerId,
		controllerLTPK: controllerLTPK,
		controllerLTSK: controllerLTSK,
		ee:             emitter.Emitter{},
	}

	//d.Use("*", emitter.Sync)
	// TODO emitter use channels

	return d
}

func (d *Device) doRequest(req *http.Request) (*http.Response, error) {
	if d.httpc == nil || d.cc.closed {
		return nil, errors.New("no http client available")
	}
	return d.httpc.Do(req)
}
func (d *Device) doPost(url string, contentType string, body io.Reader) (*http.Response, error) {
	if d.httpc == nil || d.cc.closed {
		return nil, errors.New("no http client available")
	}
	return d.httpc.Post(url, contentType, body)
}
func (d *Device) doGet(url string) (*http.Response, error) {
	if d.httpc == nil || d.cc.closed {
		return nil, errors.New("no http client available")
	}
	return d.httpc.Get(url)
}

func (d *Device) emit(topic string, args ...interface{}) {
	d.ee.Emit(topic, args...)
}

func (d *Device) OnConnect() <-chan emitter.Event {
	return d.ee.On("connect")
}
func (d *Device) OnClose() <-chan emitter.Event {
	return d.ee.On("close")
}
func (d *Device) OnPaired() <-chan emitter.Event {
	return d.ee.On("paired")
}
func (d *Device) OnVerified() <-chan emitter.Event {
	return d.ee.On("verified")
}
func (d *Device) OnUnpaired() <-chan emitter.Event {
	return d.ee.On("unpaired")
}

func (d *Device) close() error {
	var err error
	if d.cc != nil {
		d.cc.closed = true
		err = d.cc.Conn.Close()
	}
	d.verified = false
	d.httpc = nil

	d.emit("close")
	return err
}

func (d *Device) connect() error {

	if d.cc != nil {
		d.cc.Conn.Close()
	}
	d.verified = false

	dial, err := net.DialTimeout("tcp", d.tcpAddr, time.Second)
	if err != nil {
		return err
	}

	// connection, http client
	cc := newConn(dial)
	d.cc = cc
	d.httpc = &http.Client{
		Transport: newRoundTripper(d),
	}
	d.cc.SetEventCallback(d.onEvent)

	d.emit("connect")

	return nil
}

func (d *Device) startBackgroundRead() {
	d.cc.inBackground = true
	go func() {
		d.cc.loop()
		d.close()
	}()
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

// Accessories return list of previously discovered accessories.
// GetAccessories should be called prior to this call.
func (d *Device) Accessories() []*Accessory {
	return d.accs.Accs
}

// GetAccessories sends GET /accessories request and store
// result that can be retrieved with Accessories() method.
func (d *Device) GetAccessories() error {

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

// GetCharacteristic sends GET /characteristic request and return characteristic description and value.
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

// PutCharacteristic makes PUT /characteristic request to control characteristic value.
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

		topic := fmt.Sprintf("event %d %d", aid, iid)
		d.emit(topic, aid, iid, val)
	}
}

func (d *Device) SubscribeToEvents(aid uint64, iid uint64) (<-chan emitter.Event, error) {
	topic := fmt.Sprintf("event %d %d", aid, iid)

	for _, tt := range d.ee.Topics() {
		if tt == topic {
			// already subscribed
			// support multiple listeners
			return d.ee.On(topic), nil
		}
	}

	type putPayload struct {
		Cs []CharacteristicPut `json:"characteristics"`
	}

	ev := true
	c := putPayload{Cs: []CharacteristicPut{{Aid: aid, Iid: iid, Events: &ev}}}
	b, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", "/characteristics", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	res, err := d.doRequest(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusNoContent {
		return nil, errors.New("not 204")
	}

	return d.ee.On(topic), nil
}

func (d *Device) UnsubscribeFromEvents(aid uint64, iid uint64, channels ...<-chan emitter.Event) error {

	topic := fmt.Sprintf("event %d %d", aid, iid)

	if len(channels) < len(d.ee.Listeners(topic)) {
		// somebody else subscribed
		d.ee.Off(topic, channels...)
		return nil
	}

	type putPayload struct {
		Cs []CharacteristicPut `json:"characteristics"`
	}

	ev := false
	c := putPayload{Cs: []CharacteristicPut{{Aid: aid, Iid: iid, Events: &ev}}}
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

	d.ee.Off(topic)

	return nil
}
