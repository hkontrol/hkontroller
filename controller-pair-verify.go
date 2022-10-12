package hkontroller

import (
	"bytes"
	"errors"
	"github.com/hkontrol/hkontroller/chacha20poly1305"
	"github.com/hkontrol/hkontroller/curve25519"
	"github.com/hkontrol/hkontroller/ed25519"
	"github.com/hkontrol/hkontroller/hkdf"
	"github.com/hkontrol/hkontroller/tlv8"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
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

func (c *Controller) PairVerify(devId string) error {

	c.mu.Lock()
	defer c.mu.Unlock()
	pc, ok := c.devices[devId]
	if !ok {
		return errors.New("no devices accessory found")
	}
	_, ok = c.mdnsDiscovered[devId]
	if !ok {
		return errors.New("no dnssd entry found")
	}

	if pc.httpc == nil {
		// tcp conn open
		dial, err := net.Dial("tcp", pc.tcpAddr)
		if err != nil {
			return err
		}
		// connection, http client
		cc := newConn(dial)

		pc.httpc = &http.Client{
			Transport: cc,
		}
		pc.cc = cc
	}

	localPublic, localPrivate := curve25519.GenerateKeyPair()

	m1 := pairVerifyM1Payload{
		Method:    0,
		State:     M1,
		PublicKey: localPublic[:],
	}
	b, err := tlv8.Marshal(m1)
	if err != nil {
		return err
	}

	// send req
	response, err := pc.httpc.Post("/pair-verify", HTTPContentTypePairingTLV8, bytes.NewReader(b))
	if err != nil {
		return err
	}
	res := response.Body
	defer res.Close()
	all, err := ioutil.ReadAll(res)
	if err != nil {
		return err
	}
	m2 := pairSetupPayload{}
	err = tlv8.Unmarshal(all, &m2)
	if err != nil {
		return err
	}
	if m2.State != M2 {
		return errors.New("expected state M2")
	}
	if m2.PublicKey == nil || m2.EncryptedData == nil || m2.Error != 0x00 {
		return errors.New("m2err = " + strconv.FormatInt(int64(m2.Error), 10))
	}
	if len(m2.PublicKey) != 32 {
		return errors.New("wrong remote localPublic key length")
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
		return nil
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
		return err
	}
	m2dec := pairSetupPayload{}
	err = tlv8.UnmarshalReader(bytes.NewReader(decryptedBytes), &m2dec)
	if err != nil {
		return err
	}
	if m2dec.Signature == nil {
		return errors.New("no signature from accessory")
	}
	// m2.Signature
	// Validate signature
	var material []byte
	material = append(material, remotePubk[:]...)
	material = append(material, m2dec.Identifier...)
	material = append(material, localPublic[:]...)

	ltpk := pc.pairing.PublicKey

	sigValid := ed25519.ValidateSignature(ltpk, material, m2dec.Signature)
	if !sigValid {
		return errors.New("signature invalid")
	}

	// ----- M3 ------

	material = []byte{}
	material = append(material, localPublic[:]...)
	material = append(material, c.name...)
	material = append(material, remotePubk[:]...)

	signature, err := ed25519.Signature(c.localLTSK, material)
	if err != nil {
		return err
	}

	m3raw := pairVerifyM3RawPayload{
		Signature:  signature,
		Identifier: c.name,
	}
	m3bytes, err := tlv8.Marshal(m3raw)
	if err != nil {
		return err
	}

	encryptedBytes, mac, err := chacha20poly1305.EncryptAndSeal(
		encKey[:],
		[]byte("PV-Msg03"),
		m3bytes,
		nil,
	)
	if err != nil {
		return err
	}

	m5enc := pairVerifyM3EncPayload{
		State:         M3,
		EncryptedData: append(encryptedBytes, mac[:]...),
	}
	b, err = tlv8.Marshal(m5enc)
	if err != nil {
		return err
	}

	response, err = pc.httpc.Post("/pair-verify", HTTPContentTypePairingTLV8, bytes.NewReader(b))
	if err != nil {
		return err
	}
	res = response.Body

	defer res.Close()
	all, err = ioutil.ReadAll(res)
	if err != nil {
		return err
	}
	m4 := pairSetupPayload{}
	err = tlv8.Unmarshal(all, &m4)
	if err != nil {
		return err
	}

	if m4.Error != 0x00 {
		return errors.New("m4err = " + strconv.FormatInt(int64(m4.Error), 10))
	}

	//pc.httpc.CloseIdleConnections()

	ss, err := newControllerSession(sharedKey, *pc)
	if err != nil {
		return err
	}
	pc.ss = ss
	pc.cc.UpgradeEnc(ss)
	pc.verified = true

	return nil
}
