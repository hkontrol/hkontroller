package hkontroller

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hkontrol/hkontroller/log"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/hkontrol/dnssd"
	"github.com/olebedev/emitter"
)

const dialTimeout = 5 * time.Second
const reqTimeout = 15 * time.Second
const emitTimeout = 30 * time.Second

type Device struct {
	ee emitter.Emitter

	Name string

	dnssdBrowseEntry *dnssd.BrowseEntry

	controllerId   string
	controllerLTPK []byte
	controllerLTSK []byte

	pairing Pairing

	discovered bool // discovered mdns?

	paired   bool // completed /pair-setup?
	verified bool // is connection established after /pair-verify?

	closeReason error

	cc    *conn
	ss    *session
	httpc *http.Client // http client with encryption support
	accs  []*Accessory
}

type roundTripper struct {
	d  *Device
	mu sync.Mutex
}

func newRoundTripper(d *Device) *roundTripper {
	return &roundTripper{d: d, mu: sync.Mutex{}}
}

// RoundTrip implementation to be able to use with http.Client
func (r *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {

	r.mu.Lock()
	defer r.mu.Unlock()

	//fmt.Println("===== roundTrip =====")
	//dump, err := httputil.DumpRequest(req, false)
	//if err == nil {
	//	fmt.Println(string(dump))
	//}

	// make response channel
	r.d.cc.response = make(chan *http.Response)
	defer close(r.d.cc.response)
	r.d.cc.resError = make(chan error)
	defer close(r.d.cc.resError)
	r.d.cc.wantResponse = true
	defer func() { r.d.cc.wantResponse = false }()

	err := req.Write(r.d.cc)
	if err != nil {
		return nil, err
	}

	if r.d.cc.inBackground {
		// select err or response
		select {
		case res := <-r.d.cc.response:
			//fmt.Println("case <- d.cc.response")
			return res, err
		case err := <-r.d.cc.resError:
			//fmt.Println("case <- d.cc.resError")
			return nil, err
		case <-req.Context().Done():
			//fmt.Println("case <- req.Context().Done()")
			return nil, req.Context().Err()
		}
	}

	rd := bufio.NewReader(r.d.cc)
	res, err := http.ReadResponse(rd, req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func newDevice(dnssdEntry *dnssd.BrowseEntry, name string,
	controllerId string, controllerLTPK []byte, controllerLTSK []byte) *Device {

	d := &Device{
		dnssdBrowseEntry: dnssdEntry,
		Name:             name,
		controllerId:     controllerId,
		controllerLTPK:   controllerLTPK,
		controllerLTSK:   controllerLTSK,
		ee:               emitter.Emitter{},
	}

	if dnssdEntry != nil {
		d.Name = dnssdEntry.Name
	}

	return d
}

func (d *Device) setDnssdEntry(e *dnssd.BrowseEntry) {
	d.dnssdBrowseEntry = e
	if e != nil {
		d.Name = e.Name
	}
}

func (d *Device) mergeDnssdEntry(e dnssd.BrowseEntry) {
	if d.dnssdBrowseEntry == nil {
		d.dnssdBrowseEntry = &e
		return
	}
	for _, ip := range e.IPs {
		d.dnssdBrowseEntry.IPs = append(d.dnssdBrowseEntry.IPs, ip)
	}
}

func (d *Device) GetDnssdEntry() dnssd.BrowseEntry {
	if d.dnssdBrowseEntry != nil {
		return *d.dnssdBrowseEntry
	}
	return dnssd.BrowseEntry{}
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
	defer func() {
		// sometimes done channel is closed before emitTimeout
		recover()
	}()
	done := d.ee.Emit(topic, args...)
	select {
	case <-done:
		// so the sending is done
	case <-time.After(emitTimeout):
		// time is out, let's discard emitting
		close(done)
	}
}

func (d *Device) offAllTopics() {
	for _, t := range d.ee.Topics() {
		d.ee.Off(t)
	}
}
func (d *Device) OnDiscovered() <-chan emitter.Event {
	return d.ee.On("discover")
}
func (d *Device) OffDiscovered(ch <-chan emitter.Event) {
	d.ee.Off("discovered", ch)
}
func (d *Device) OnLost() <-chan emitter.Event {
	return d.ee.On("lost")
}
func (d *Device) OffLost(ch <-chan emitter.Event) {
	d.ee.Off("lost", ch)
}
func (d *Device) OnClose() <-chan emitter.Event {
	return d.ee.On("close")
}
func (d *Device) OffClose(ch <-chan emitter.Event) {
	d.ee.Off("close", ch)
}
func (d *Device) OnPaired() <-chan emitter.Event {
	return d.ee.On("paired")
}
func (d *Device) OffPaired(ch <-chan emitter.Event) {
	d.ee.Off("paired", ch)
}
func (d *Device) OnVerified() <-chan emitter.Event {
	return d.ee.On("verified")
}
func (d *Device) OffVerified(ch <-chan emitter.Event) {
	d.ee.Off("verified", ch)
}
func (d *Device) OnUnpaired() <-chan emitter.Event {
	return d.ee.On("unpaired")
}
func (d *Device) OffUnpaired(ch <-chan emitter.Event) {
	d.ee.Off("unpaired", ch)
}

func (d *Device) close(reason error) error {
	log.Debug.Println("device <%s> close call with reason: %v\n", d.Name, reason)
	d.closeReason = reason
	var err error
	if d.cc != nil {
		d.cc.close()
		if d.verified {
			for range d.cc.backgroundStop {
			}
		}
		d.cc = nil
	}
	d.verified = false
	d.httpc = nil

	d.accs = nil

	d.ee.Off("event*") // close all subscriptions to char events

	d.emit("close")
	return err
}

func (d *Device) Close() error {
	return d.close(errors.New("manual close "))
}

func (d *Device) connect() error {

	if d.cc != nil {
		d.close(errors.New("close on reconnect"))
	}

	if d.dnssdBrowseEntry == nil || !d.discovered {
		return errors.New("not discovered")
	}

	dial, err := dialServiceInstance(context.Background(), d.dnssdBrowseEntry, dialTimeout)
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

	d.closeReason = nil

	return nil
}

func (d *Device) startBackgroundRead() {
	go func() {
		d.cc.loop()
		log.Debug.Println("background read: loop stopped")
		d.close(errors.New("stop background read"))
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

func (d *Device) GetPairingInfo() Pairing {
	return d.pairing
}

// IsVerified returns true if /pair-verify step was completed by this controller.
func (d *Device) IsVerified() bool {
	return d.verified
}

// CloseReason returns last close reason if connection is closed
func (d *Device) CloseReason() error {
	return d.closeReason
}

// Accessories return list of previously discovered accessories.
// GetAccessories should be called prior to this call.
func (d *Device) Accessories() []*Accessory {
	return d.accs
}

// GetAccessories sends GET /accessories request and store
// result that can be retrieved with Accessories() method.
func (d *Device) GetAccessories() error {

	if !d.verified || d.httpc == nil {
		return errors.New("paired device not verified or not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), reqTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "/accessories", nil)
	if err != nil {
		return err
	}
	res, err := d.doRequest(req)
	if err != nil {
		return nil
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

	// shorten UUIDs
	for _, a := range accs.Accs {
		for _, s := range a.Ss {
			s.Type = s.Type.ToShort()
			for _, c := range s.Cs {
				c.Type = c.Type.ToShort()
			}
		}
	}

	d.accs = accs.Accs

	return nil
}

// GetCharacteristic sends GET /characteristic request and return characteristic description and value.
func (d *Device) GetCharacteristic(aid uint64, cid uint64) (CharacteristicDescription, error) {
	ep := fmt.Sprintf("/characteristics?id=%d.%d", aid, cid)

	ctx, cancel := context.WithTimeout(context.Background(), reqTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", ep, nil)

	if err != nil {
		return CharacteristicDescription{}, err
	}

	res, err := d.doRequest(req)
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

	ctx, cancel := context.WithTimeout(context.Background(), reqTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "PUT", "/characteristics", bytes.NewReader(b))
	if err != nil {
		return err
	}

	res, err := d.doRequest(req)
	if err != nil {
		return err
	}
	_ = res

	//all, err := io.ReadAll(res.Body)
	//if err != nil {
	//	return err
	//}
	//
	// TODO: extract errors. if response message is empty then no error occured
	// 		 case of error:
	//        {"characteristics":[{"aid":1,"iid":12,"status":-70402}]}
	// fmt.Println(string(all))

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

		// update values, so it will be available without extra GetAccessories request
		for _, aa := range d.Accessories() {
			if aa.Id != aid {
				continue
			}
			for _, ss := range aa.Ss {
				for _, cc := range ss.Cs {
					if cc.Iid == iid {
						cc.Value = val
					}
				}
			}
		}

		topic := fmt.Sprintf("event %d %d", aid, iid)
		d.emit(topic, aid, iid, val)
	}
}

func (d *Device) getEventPutPayloadForAccessory(aid int, subscribe bool) []CharacteristicPut {
	var cs []CharacteristicPut
	for _, aa := range d.Accessories() {
		if int(aa.Id) != aid && aid > -1 { // if aid < 0 we get list for all accs
			continue
		}
		for _, ss := range aa.Ss {
			for _, cc := range ss.Cs {
				for _, pp := range cc.Permissions {
					if pp == "ev" { // supported
						ev := subscribe
						cs = append(cs, CharacteristicPut{Aid: aa.Id, Iid: cc.Iid, Events: &ev})
					}
				}
			}
		}
	}
	return cs
}

func (d *Device) SubscribeToEvents(aid uint64, iid uint64) (<-chan emitter.Event, error) {
	topic := fmt.Sprintf("event %d %d", aid, iid)
	super := fmt.Sprintf("event %d *", aid)
	mega := fmt.Sprintf("event * *")

	for _, tt := range d.ee.Topics() {
		if tt == topic || tt == super || tt == mega {
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

func (d *Device) SubscribeToAccessoryEvents(aid uint64) (<-chan emitter.Event, error) {

	topic := fmt.Sprintf("event %d *", aid)
	super := fmt.Sprintf("event * *")

	for _, tt := range d.ee.Topics() {
		if tt == topic || tt == super {
			// already subscribed
			// support multiple listeners
			return d.ee.On(topic), nil
		}
	}

	type putPayload struct {
		Cs []CharacteristicPut `json:"characteristics"`
	}
	pl := putPayload{Cs: d.getEventPutPayloadForAccessory(int(aid), true)}

	b, err := json.Marshal(pl)
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

func (d *Device) SubscribeToAllEvents() (<-chan emitter.Event, error) {

	topic := fmt.Sprintf("event * *")

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

	pl := putPayload{Cs: d.getEventPutPayloadForAccessory(-1, true)}

	b, err := json.Marshal(pl)
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

func (d *Device) UnsubscribeFromAllEvents(channels ...<-chan emitter.Event) error {

	topic := fmt.Sprintf("event * *")

	if len(channels) != 0 && len(channels) < len(d.ee.Listeners(topic)) {
		// somebody else subscribed
		d.ee.Off(topic, channels...)
		return nil
	}

	type putPayload struct {
		Cs []CharacteristicPut `json:"characteristics"`
	}

	pl := putPayload{Cs: d.getEventPutPayloadForAccessory(-1, false)}

	b, err := json.Marshal(pl)
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

	// close all related channels
	for _, pp := range pl.Cs {
		topic := fmt.Sprintf("event %d %d", pp.Aid, pp.Iid)
		d.ee.Off(topic)
	}

	return nil
}

func (d *Device) UnsubscribeFromAccessoryEvents(aid uint64, channels ...<-chan emitter.Event) error {

	topic := fmt.Sprintf("event %d *", aid)

	if len(channels) != 0 && len(channels) < len(d.ee.Listeners(topic)) {
		// somebody else subscribed
		d.ee.Off(topic, channels...)
		return nil
	}

	type putPayload struct {
		Cs []CharacteristicPut `json:"characteristics"`
	}

	pl := putPayload{Cs: d.getEventPutPayloadForAccessory(-1, false)}

	b, err := json.Marshal(pl)
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

	// close all related channels
	for _, pp := range pl.Cs {
		topic := fmt.Sprintf("event %d %d", pp.Aid, pp.Iid)
		d.ee.Off(topic)
	}

	return nil
}

func (d *Device) UnsubscribeFromEvents(aid uint64, iid uint64, channels ...<-chan emitter.Event) error {

	topic := fmt.Sprintf("event %d %d", aid, iid)

	if len(channels) != 0 && len(channels) < len(d.ee.Listeners(topic)) {
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
