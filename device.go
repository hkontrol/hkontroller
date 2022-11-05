package hkontroller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

type Device struct {
	Id string

	pairing Pairing

	discovered bool // discovered via mdns?
	paired     bool // completed /pair-setup?
	verified   bool // is connection established after /pair-verify?

	tcpAddr  string // tcp socket address
	httpAddr string // tcp socket address
	cc       *conn
	ss       *session
	httpc    *http.Client // http client with encryption support
	accs     *Accessories
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

	res, err := d.httpc.Get("/accessories")
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
	res, err := d.httpc.Get(ep)
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

	/***
	PUT /characteristics HTTP/1.1
	Host: lights.local:12345
	Content-Type: application/hap+json
	Content-Length: <length>
	{
	    ”characteristics” :
	    [{
	        ”aid” : 2,
	        ”iid” : 8,
	        ”value” : true
	    },...]
	}
	 ***/

	type putPayload struct {
		Cs []CharacteristicPut `json:"characteristics"`
	}

	c := putPayload{Cs: []CharacteristicPut{{Aid: aid, Iid: cid, Value: val}}}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	fmt.Println("marshalled: ", string(b))

	req, err := http.NewRequest("PUT", "/characteristics", bytes.NewReader(b))
	if err != nil {
		return err
	}

	_, err = d.httpc.Do(req)
	if err != nil {
		return err
	}

	return nil
}

func (d *Device) SubscribeToEvents(ctx context.Context, aid uint64, cid uint64) error {

	/***
	PUT /characteristics HTTP/1.1
	Host: lights.local:12345
	Content-Type: application/hap+json
	Content-Length: <length>
	{
	    ”characteristics” :
	    [{
	        ”aid” : 2,
	        ”iid” : 8,
	        "ev": true
	    },...]
	}
	 ***/

	type putPayload struct {
		Cs []CharacteristicPut `json:"characteristics"`
	}

	ev := true
	c := putPayload{Cs: []CharacteristicPut{{Aid: aid, Iid: cid, Events: &ev}}}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	fmt.Println("marshalled: ", string(b))

	req, err := http.NewRequest("PUT", "/characteristics", bytes.NewReader(b))
	if err != nil {
		return err
	}

	_, err = d.httpc.Do(req)
	if err != nil {
		return err
	}

	return nil
}
