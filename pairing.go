package hkontrol

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
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
	p.accs = &accs

	return nil
}

func (p *Pairing) GetAccessories() []*Accessory {
	return p.accs.Accs
}
