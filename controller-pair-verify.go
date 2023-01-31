package hkontroller

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/hkontrol/hkontroller/chacha20poly1305"
	"github.com/hkontrol/hkontroller/curve25519"
	"github.com/hkontrol/hkontroller/ed25519"
	"github.com/hkontrol/hkontroller/hkdf"
	"github.com/hkontrol/hkontroller/log"
	"github.com/hkontrol/hkontroller/tlv8"
)

type pairVerifyM1Payload struct {
	Method    byte   `tlv8:"0"`
	State     byte   `tlv8:"6"`
	PublicKey []byte `tlv8:"3"`
}
type pairVerifyM3RawPayload struct {
	Identifier string `tlv8:"1"`
	Signature  []byte `tlv8:"10"`
}
type pairVerifyM3EncPayload struct {
	State         byte   `tlv8:"6"`
	EncryptedData []byte `tlv8:"5"`
}

// PairVerify
func (d *Device) PairVerify() error {
	if !d.paired {
		return errors.New("not paired before verifying")
	}
	if d.cc == nil {
		err := d.connect()
		if err != nil {
			return err
		}
	}
	if d.cc.closed {
		err := d.connect()
		if err != nil {
			return err
		}
	}

	localPublic, localPrivate := curve25519.GenerateKeyPair()

	m1 := pairVerifyM1Payload{
		Method:    0,
		State:     M1,
		PublicKey: localPublic[:],
	}
	b, err := tlv8.Marshal(m1)
	if err != nil {
		return &PairVerifyError{"M1", err}
	}

	// send req
	response, err := d.doPost("/pair-verify", HTTPContentTypePairingTLV8, bytes.NewReader(b))
	if err != nil {
		return &PairVerifyError{"M1", err}
	}
	res := response.Body
	defer res.Close()
	all, err := io.ReadAll(res)
	if err != nil {
		return &PairVerifyError{"M2", err}
	}
	m2 := pairSetupPayload{}
	err = tlv8.Unmarshal(all, &m2)
	if err != nil {
		return &PairVerifyError{"M2", err}
	}
	if m2.State != M2 {
		return &PairVerifyError{"M2", fmt.Errorf("unexpected state %x, expected: %x", m2.State, M2)}
	}
	if m2.PublicKey == nil || m2.EncryptedData == nil || m2.Error != 0x00 {
		return &PairVerifyError{"M2", TlvErrorFromCode(m2.Error)}
	}
	if len(m2.PublicKey) != 32 {
		return &PairVerifyError{"M2", errors.New("wrong remote localPublic key length")}
	}
	remotePubk := [32]byte{}
	copy(remotePubk[:], m2.PublicKey)

	sharedKey := curve25519.SharedSecret(localPrivate, remotePubk)

	encKey, err := hkdf.Sha512(
		sharedKey[:],
		[]byte("Pair-Verify-Encrypt-Salt"),
		[]byte("Pair-Verify-Encrypt-Info"),
	)
	if err != nil {
		return &PairVerifyError{"M2", err}
	}

	data := m2.EncryptedData
	message := data[:(len(data) - 16)]
	var mac [16]byte
	copy(mac[:], data[len(message):]) // 16 byte (MAC)

	decryptedBytes, err := chacha20poly1305.DecryptAndVerify(
		encKey[:],
		[]byte("PV-Msg02"),
		message,
		mac,
		nil,
	)
	if err != nil {
		return &PairVerifyError{"M2", err}
	}
	m2dec := pairSetupPayload{}
	err = tlv8.UnmarshalReader(bytes.NewReader(decryptedBytes), &m2dec)
	if err != nil {
		return &PairVerifyError{"M2", err}
	}
	if m2dec.Signature == nil {
		return &PairVerifyError{"M2", errors.New("no signature from accessory")}
	}
	// m2.Signature
	// Validate signature
	var material []byte
	material = append(material, remotePubk[:]...)
	material = append(material, m2dec.Identifier...)
	material = append(material, localPublic[:]...)

	ltpk := d.pairing.PublicKey

	sigValid := ed25519.ValidateSignature(ltpk, material, m2dec.Signature)
	if !sigValid {
		return &PairVerifyError{"M2", errors.New("signature invalid")}
	}

	// ----- M3 ------

	material = []byte{}
	material = append(material, localPublic[:]...)
	material = append(material, d.controllerId...)
	material = append(material, remotePubk[:]...)

	signature, err := ed25519.Signature(d.controllerLTSK, material)
	if err != nil {
		return &PairVerifyError{"M3", err}
	}

	m3raw := pairVerifyM3RawPayload{
		Signature:  signature,
		Identifier: d.controllerId,
	}
	m3bytes, err := tlv8.Marshal(m3raw)
	if err != nil {
		return &PairVerifyError{"M3", err}
	}

	encryptedBytes, mac, err := chacha20poly1305.EncryptAndSeal(
		encKey[:],
		[]byte("PV-Msg03"),
		m3bytes,
		nil,
	)
	if err != nil {
		return &PairVerifyError{"M3", err}
	}

	m5enc := pairVerifyM3EncPayload{
		State:         M3,
		EncryptedData: append(encryptedBytes, mac[:]...),
	}
	b, err = tlv8.Marshal(m5enc)
	if err != nil {
		return &PairVerifyError{"M3", err}
	}

	response, err = d.doPost("/pair-verify", HTTPContentTypePairingTLV8, bytes.NewReader(b))
	if err != nil {
		return &PairVerifyError{"M4", err}
	}
	res = response.Body

	defer res.Close()
	all, err = io.ReadAll(res)
	if err != nil {
		return &PairVerifyError{"M4", err}
	}
	m4 := pairSetupPayload{}
	err = tlv8.Unmarshal(all, &m4)
	if err != nil {
		return &PairVerifyError{"M4", err}
	}

	if m4.Error != 0x00 {
		return &PairVerifyError{"M4", TlvErrorFromCode(m4.Error)}
	}

	ss, err := newControllerSession(sharedKey, d)
	if err != nil {
		return &PairVerifyError{"M4", err}
	}
	d.ss = ss
	d.cc.UpgradeEnc(ss)
	d.verified = true

	d.startBackgroundRead()

	d.emit("verified")

	return nil
}

// PairSetupAndVerify first setup pairing if was not set before
// then establish encrypted connection
// that should automatically reconnect in case of failure.
func (d *Device) PairSetupAndVerify(ctx context.Context, pin string, retryTimeout time.Duration) error {
	var err error

	// pair-setup should be done in any case
	if !d.paired {
		err = d.PairSetup(pin)
	}
	if err != nil {
		return err
	}

	// then encrypted channel should be persisted
	go d.pairVerifyPersist(ctx, retryTimeout)

	verifiedEv := d.OnVerified()
	unpairedEv := d.OnUnpaired()
	lostEv := d.OnLost()
	defer func() {
		d.OffVerified(verifiedEv)
		d.OffUnpaired(unpairedEv)
		d.OffLost(lostEv)
	}()
	select {
	case <-verifiedEv:
		return nil
	case <-lostEv:
		return errors.New("device lost")
	case <-unpairedEv:
		return errors.New("unpaired")
	}
}

// pairVerifyPersist establish encrypted connection with auto-reconnect.
// Connection broke if device is unpaired. May be cancelled by context as well.
func (d *Device) pairVerifyPersist(ctx context.Context, retryTimeout time.Duration) error {
	newCtx, cancel := context.WithCancel(ctx)
	d.cancelPersistConnection = cancel
	errorEv := d.OnError()
	unpairedEv := d.OnUnpaired()
	closedEv := d.OnClose()
	lostEv := d.OnLost() // mdns lost
	defer func() {
		d.OffError(errorEv)
		d.OffClose(closedEv)
		d.OffUnpaired(unpairedEv)
		d.OffLost(lostEv)
		d.close()
	}()
	for {
		go func() {
			if !d.discovered || d.dnssdBrowseEntry == nil {
				//d.emit("lost")
				return
			}
			if d.paired && !d.verified {
				err := d.PairVerify()
				if err != nil {
					// just emit error in this case
					// sometimes error may occur if connection was broken, etc
					// so no need to unpair
					d.emit("error", err)
					return
				}
			} else if d.paired && d.verified {
				// to catch later
				d.emit("verified")
			}
		}()

		// catch events
		select {
		case ev := <-errorEv:
			log.Debug.Println("error event")
			if len(ev.Args) > 0 {
				// check error - if it's protocol error, e.g. invalid signature, then return err
				if err, ok := ev.Args[0].(error); ok {
					var pSetupErr *PairSetupError
					var pVerifyErr *PairVerifyError
					if errors.As(err, &pSetupErr) {
						return err
					}
					if errors.As(err, &pVerifyErr) {
						return err
					}
				}
			}
			time.Sleep(retryTimeout)
			// reconnect
			continue
		case <-closedEv:
			log.Debug.Println("close event")
			select {
			case <-time.After(retryTimeout):
				// connection was closed, back to reconnect loop
				continue
			case <-lostEv:
				// connection was closed and device is not advertising itself no more
				log.Debug.Println("lost event")
				return errors.New("lost")
			}
		case <-unpairedEv:
			d.cancelPersistConnection = nil
			log.Debug.Println("unpaired event")
			// in this case we don't need connection anymore
			return errors.New("unpaired")
		case <-newCtx.Done():
			d.cancelPersistConnection = nil
			d.close()
			return errors.New("cancelled")
		}
	}
}
