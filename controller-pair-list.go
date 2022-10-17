package hkontroller

import (
	"bytes"
	"fmt"
	"github.com/hkontrol/hkontroller/tlv8"
	"io"
	"net/http"
)

type pairListReqPayload struct {
	Method byte `tlv8:"0"`
	State  byte `tlv8:"6"`
}
type pairingPayload struct {
	Identifier string `tlv8:"1"`
	PublicKey  []byte `tlv8:"3"`
	Permission byte   `tlv8:"11"`
}

// ListPairings should list all controllers of device.
// Currently doesn't work as expected
func (d *Device) ListPairings() ([]Pairing, error) {

	pl := pairListReqPayload{
		State:  M1,
		Method: MethodListPairings,
	}
	b, err := tlv8.Marshal(pl)
	if err != nil {
		return nil, err
	}

	ep := fmt.Sprintf("%s/%s", d.httpAddr, "pairings")

	resp, err := d.httpc.Post(ep, HTTPContentTypePairingTLV8, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid status code %v", resp.StatusCode)
	}
	res := resp.Body
	defer res.Close()
	all, err := io.ReadAll(res)
	if err != nil {
		return nil, err
	}
	m2 := pairingPayload{} // TODO examine
	err = tlv8.Unmarshal(all, &m2)
	// if one controller is paired, there is no need for []pairingPayload
	// but what if there is multiple pairings?
	// tlv8.Unmarshal do creates empty slice

	if err != nil {
		return nil, err
	}

	var result []Pairing
	result = append(result, Pairing{Name: m2.Identifier, PublicKey: m2.PublicKey, Permission: m2.Permission})

	return result, nil
}
