package hkontroller

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
)

// Pairing is the pairing of a controller with the server.
type Pairing struct {
	Name      string `json:"name"`
	PublicKey []byte `json:"publicKey"`

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

func (p *Pairing) DiscoverAccessories() error {

	if !p.verified || p.httpc == nil {
		return errors.New("paired device not verified or not connected")
	}

	res, err := p.httpc.Get("/accessories")
	if err != nil {
		return err
	}
	all, err := ioutil.ReadAll(res.Body)
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

	p.accs = &accs

	return nil
}

func (p *Pairing) GetAccessories() []*Accessory {
	return p.accs.Accs
}

func (p *Pairing) PutCharacteristic(aid uint64, cid uint64, val interface{}) error {

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

	_, err = p.httpc.Do(req)
	if err != nil {
		return err
	}

	return nil
}
