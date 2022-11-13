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

type pairRemoveReqPayload struct {
	State      byte   `tlv8:"6"`
	Method     byte   `tlv8:"0"`
	Identifier string `tlv8:"1"`
}

type pairRemoveResPayload struct {
	State byte `tlv8:"6"`
	Error byte `tlv8:"7"`
}

func (d *Device) Unpair() error {

	pl := pairRemoveReqPayload{
		State:      M1,
		Method:     MethodDeletePairing,
		Identifier: d.controllerId,
	}
	b, err := tlv8.Marshal(pl)
	if err != nil {
		return err
	}

	defer func() {
		d.cc.Close()
		d.httpc = nil
		d.cc = nil
		d.paired = false
		d.verified = false
		d.httpc = nil
		d.Emit("unpaired")
		d.Emit("close")
	}()

	resp, err := d.doPost("/pairings", HTTPContentTypePairingTLV8, bytes.NewReader(b))
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
	fmt.Println("unpairResponse: ", string(all))
	m2 := pairRemoveResPayload{}
	err = tlv8.Unmarshal(all, &m2)

	if m2.Error != 0x00 {
		return errors.New("res.err = " + strconv.FormatInt(int64(m2.Error), 10))
	}

	if err != nil {
		return err
	}

	return nil
}
