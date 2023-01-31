package hkontroller

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/hkontrol/hkontroller/chacha20poly1305"
	"github.com/hkontrol/hkontroller/ed25519"
	"github.com/hkontrol/hkontroller/hkdf"
	"github.com/hkontrol/hkontroller/tlv8"
	"io"
	"net/http"
)

type pairSetupM1Payload struct {
	Method byte `tlv8:"0"`
	State  byte `tlv8:"6"`
}
type pairSetupM3Payload struct {
	PublicKey []byte `tlv8:"3"`
	Proof     []byte `tlv8:"4"`
	State     byte   `tlv8:"6"`
}
type pairSetupM5RawPayload struct {
	Identifier string `tlv8:"1"`
	PublicKey  []byte `tlv8:"3"`
	Signature  []byte `tlv8:"10"`
}
type pairSetupM5EncPayload struct {
	Method        byte   `tlv8:"0"`
	State         byte   `tlv8:"6"`
	EncryptedData []byte `tlv8:"5"`
}

func (d *Device) pairSetupM1(pin string) (*pairSetupClientSession, error) {

	m1 := pairSetupM1Payload{
		State:  M1,
		Method: MethodPair,
	}
	b, err := tlv8.Marshal(m1)
	if err != nil {
		return nil, &PairSetupError{"M1", err}
	}

	resp, err := d.doPost("/pair-setup", HTTPContentTypePairingTLV8, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &PairSetupError{"M1", fmt.Errorf("invalid status code %v", resp.StatusCode)}
	}
	res := resp.Body
	defer res.Close()
	all, err := io.ReadAll(res)
	if err != nil {
		return nil, &PairSetupError{"M1", err}
	}
	m2 := pairSetupPayload{}
	err = tlv8.Unmarshal(all, &m2)
	if err != nil {
		return nil, &PairSetupError{"M1", err}
	}

	salt := m2.Salt
	_ = salt
	remotePubk := m2.PublicKey
	_ = remotePubk
	state := m2.State
	m2err := m2.Error
	if state != M2 {
		return nil, &PairSetupError{"M2", fmt.Errorf("unexpected state %x, expected: %x", state, M2)}
	}
	if salt == nil && remotePubk == nil && m2err != 0x00 {
		return nil, &PairSetupError{"M2", TlvErrorFromCode(m2err)}
	}

	clientSession, err := newPairSetupClientSession(salt, remotePubk, pin)
	if err != nil {
		return nil, &PairSetupError{"M2", err}
	}

	return clientSession, nil
}

func (d *Device) pairSetupM3(clientSession *pairSetupClientSession) error {

	// m3
	m3 := pairSetupM3Payload{
		State:     M3,
		PublicKey: clientSession.PublicKey,
		Proof:     clientSession.Proof,
	}
	b, err := tlv8.Marshal(m3)
	if err != nil {
		return &PairSetupError{"M3", err}
	}

	resp, err := d.doPost("/pair-setup", HTTPContentTypePairingTLV8, bytes.NewReader(b))
	if err != nil {
		return &PairSetupError{"M3", err}
	}
	if resp.StatusCode != http.StatusOK {
		return &PairSetupError{"M3", fmt.Errorf("invalid status code %v", resp.StatusCode)}
	}
	res := resp.Body
	defer res.Close()
	all, err := io.ReadAll(res)
	if err != nil {
		return &PairSetupError{"M3", err}
	}
	m4 := pairSetupPayload{}
	err = tlv8.Unmarshal(all, &m4)
	if err != nil {
		return &PairSetupError{"M3", err}
	}

	state := m4.State
	if state != M4 {
		return &PairSetupError{"M4", fmt.Errorf("unexpected state %x, expected: %x", state, M4)}
	}
	serverProof := m4.Proof
	m4err := m4.Error
	if serverProof == nil && m4err != 0x00 {
		return &PairSetupError{"M4", TlvErrorFromCode(m4err)}
	}
	serverProofValid := clientSession.session.VerifyServerAuthenticator(serverProof)
	if !serverProofValid {
		return &PairSetupError{"M4", errors.New("server proof is not valid")}
	}

	return nil
}

func (d *Device) pairSetupM5(clientSession *pairSetupClientSession) error {

	err := clientSession.SetupEncryptionKey(
		[]byte("Pair-Setup-Encrypt-Salt"),
		[]byte("Pair-Setup-Encrypt-Info"),
	)
	if err != nil {
		return &PairSetupError{"M5", err}
	}

	hash, err := hkdf.Sha512(
		clientSession.SessionKey,
		[]byte("Pair-Setup-Controller-Sign-Salt"),
		[]byte("Pair-Setup-Controller-Sign-Info"),
	)
	if err != nil {
		return &PairSetupError{"M5", err}
	}

	var material []byte
	material = append(material, hash[:]...)
	material = append(material, d.controllerId...)
	material = append(material, d.controllerLTPK...)

	signature, err := ed25519.Signature(d.controllerLTSK, material)
	if err != nil {
		return &PairSetupError{"M5", err}
	}

	m5raw := pairSetupM5RawPayload{
		Identifier: d.controllerId,
		PublicKey:  d.controllerLTPK,
		Signature:  signature,
	}
	b, err := tlv8.Marshal(m5raw)
	if err != nil {
		return &PairSetupError{"M5", err}
	}

	encryptedBytes, tag, err := chacha20poly1305.EncryptAndSeal(
		clientSession.EncryptionKey[:],
		[]byte("PS-Msg05"),
		b,
		nil,
	)
	if err != nil {
		return &PairSetupError{"M5", err}
	}

	encData := append(encryptedBytes, tag[:]...)
	m5enc := pairSetupM5EncPayload{
		Method:        0,
		State:         M5,
		EncryptedData: encData,
	}
	b, err = tlv8.Marshal(m5enc)
	if err != nil {
		return &PairSetupError{"M5", err}
	}
	resp, err := d.doPost("/pair-setup", HTTPContentTypePairingTLV8, bytes.NewReader(b))
	if err != nil {
		return &PairSetupError{"M5", err}
	}
	if resp.StatusCode != http.StatusOK {
		return &PairSetupError{"M6", fmt.Errorf("invalid status code %v", resp.StatusCode)}
	}
	res := resp.Body
	defer res.Close()
	all, err := io.ReadAll(res)
	if err != nil {
		return &PairSetupError{"M6", err}
	}
	m6enc := pairSetupPayload{}
	err = tlv8.Unmarshal(all, &m6enc)
	if err != nil {
		return &PairSetupError{"M6", err}
	}
	if m6enc.EncryptedData == nil && m6enc.Error != 0x00 {
		return &PairSetupError{"M6", TlvErrorFromCode(m6enc.Error)}
	}

	message := m6enc.EncryptedData[:(len(m6enc.EncryptedData) - 16)]
	var mac [16]byte
	copy(mac[:], m6enc.EncryptedData[len(message):]) // 16 byte (MAC)

	decrypted, err := chacha20poly1305.DecryptAndVerify(
		clientSession.EncryptionKey[:],
		[]byte("PS-Msg06"),
		message,
		mac,
		nil,
	)

	m6dec := pairSetupPayload{}
	err = tlv8.UnmarshalReader(bytes.NewReader(decrypted), &m6dec)
	if err != nil {
		return &PairSetupError{"M6", err}
	}

	//log.Println("m6dec.State = ", m6dec.State) somehow it's state is 0
	//if m6dec.State != M6 {
	//	return errors.New("expected state M6")
	//}
	if m6dec.PublicKey == nil && m6dec.Error != 0x00 {
		return &PairSetupError{"M6", TlvErrorFromCode(m6dec.Error)}
	}

	accessoryId := m6dec.Identifier
	accessorySignature := m6dec.Signature
	accessoryLTPK := m6dec.PublicKey

	hash, err = hkdf.Sha512(
		clientSession.SessionKey,
		[]byte("Pair-Setup-Accessory-Sign-Salt"),
		[]byte("Pair-Setup-Accessory-Sign-Info"),
	)
	if err != nil {
		return &PairSetupError{"M6", err}
	}

	accessoryInfo := hash[:]
	accessoryInfo = append(accessoryInfo, accessoryId...)
	accessoryInfo = append(accessoryInfo, accessoryLTPK...)

	valid := ed25519.ValidateSignature(accessoryLTPK, accessoryInfo, accessorySignature)
	if !valid {
		return &PairSetupError{"M6", errors.New("m6 signature is not valid")}
	}

	d.pairing.Id = accessoryId
	d.pairing.PublicKey = accessoryLTPK
	//device.tcpAddr = devTcpAddr
	return nil
}

func (d *Device) PairSetup(pin string) error {

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
	// tcp conn open

	clientSession, err := d.pairSetupM1(pin)
	if err != nil {
		return err
	}
	err = d.pairSetupM3(clientSession)
	if err != nil {
		return err
	}
	err = d.pairSetupM5(clientSession)
	if err != nil {
		return err
	}
	d.paired = true
	d.verified = false
	d.emit("paired")
	return nil
}
