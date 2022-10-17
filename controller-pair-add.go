package hkontroller

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/hkontrol/hkontroller/tlv8"
	"io"
	"net/http"
	"strconv"
)

type pairAddReqPayload struct {
	State       byte   `tlv8:"6"`
	Method      byte   `tlv8:"0"`
	Identifier  string `tlv8:"1"`
	PublicKey   []byte `tlv8:"3"`
	Permissions byte   `tlv8:"11"`
}

type pairAddResPayload struct {
	State byte `tlv8:"6"`
	Error byte `tlv8:"7"`
}

// PairAdd serves to pair another controller.
func (c *Controller) PairAdd(devId string, p Pairing) error {
	c.mu.Lock()
	d, ok := c.devices[devId]
	c.mu.Unlock()
	if !ok {
		return errors.New("no device found")
	}

	pl := pairAddReqPayload{
		State:       M1,
		Method:      MethodAddPairing,
		Identifier:  p.Name,
		PublicKey:   p.PublicKey,
		Permissions: p.Permission,
	}
	b, err := tlv8.Marshal(pl)
	if err != nil {
		return err
	}

	resp, err := d.httpc.Post("/pairings", HTTPContentTypePairingTLV8, bytes.NewReader(b))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid status code %v", resp.StatusCode)
	}
	res := resp.Body
	defer res.Close()
	all, err := io.ReadAll(res)
	if err != nil {
		return err
	}
	m2 := pairAddResPayload{}
	err = tlv8.Unmarshal(all, &m2)

	if m2.Error != 0x00 {
		return errors.New("res.err = " + strconv.FormatInt(int64(m2.Error), 10))
	}

	if err != nil {
		return err
	}

	return nil
}
