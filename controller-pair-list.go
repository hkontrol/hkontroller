package hkontroller

import (
	"bytes"
	"encoding/binary"
	"errors"
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
// Currently, doesn't work as expected
func (d *Device) ListPairings() ([]Pairing, error) {

	pl := pairListReqPayload{
		State:  M1,
		Method: MethodListPairings,
	}
	b, err := tlv8.Marshal(pl)
	if err != nil {
		return nil, err
	}

	resp, err := d.httpc.Post("/pairings", HTTPContentTypePairingTLV8, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid status code %v", resp.StatusCode)
	}
	res := resp.Body
	defer res.Close()

	//
	////l := len(all)
	////o := 0
	var tag byte
	err = binary.Read(res, binary.BigEndian, &tag)
	if err != nil {
		return nil, err
	}
	var leng byte
	err = binary.Read(res, binary.BigEndian, &leng)
	if err != nil {
		return nil, err
	}
	val := make([]byte, leng)
	err = binary.Read(res, binary.BigEndian, &val)
	if err != nil {
		return nil, err
	}

	if tag != 6 && val[0] != M2 {
		return nil, errors.New("wrong response")
	}

	var result []Pairing

	var pp Pairing
	for err == nil {
		var tag byte
		err = binary.Read(res, binary.BigEndian, &tag)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		var leng byte
		err = binary.Read(res, binary.BigEndian, &leng)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if tag == 0xff {
			// separator
			result = append(result, pp)
			pp = Pairing{}
			continue
		}
		val := make([]byte, leng)
		err = binary.Read(res, binary.BigEndian, &val)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if tag == 1 {
			pp.Name = string(val)
		} else if tag == 3 {
			pp.PublicKey = val
		} else if tag == 11 {
			pp.Permission = val[0]
		}
	}
	if err == io.EOF {
		result = append(result, pp)
		err = nil
	}

	return result, nil
}
