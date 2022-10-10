package hkontrol

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/hkontrol/hkontroller/chacha20poly1305"
	"github.com/hkontrol/hkontroller/ed25519"
	"github.com/hkontrol/hkontroller/hkdf"
	"github.com/hkontrol/hkontroller/tlv8"
	"io/ioutil"
	"net/http"
	"strconv"
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

func (c *Controller) PairSetupM1(pairing *Pairing, pin string) (*pairSetupClientSession, error) {
	fmt.Println("PairSetupM1")

	ep := fmt.Sprintf("%s/%s", pairing.httpAddr, "pair-setup")

	m1 := pairSetupM1Payload{
		State:  M1,
		Method: MethodPair,
	}
	b, err := tlv8.Marshal(m1)
	if err != nil {
		return nil, err
	}

	fmt.Println("M1 sending post req")
	resp, err := pairing.httpc.Post(ep, HTTPContentTypePairingTLV8, bytes.NewReader(b))
	fmt.Println(resp, err)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid status code %v", resp.StatusCode)
	}
	res := resp.Body
	defer res.Close()
	all, err := ioutil.ReadAll(res)
	if err != nil {
		return nil, err
	}
	m2 := pairSetupPayload{}
	err = tlv8.Unmarshal(all, &m2)
	if err != nil {
		return nil, err
	}

	salt := m2.Salt
	_ = salt
	remotePubk := m2.PublicKey
	_ = remotePubk
	state := m2.State
	m2err := m2.Error
	if state != M2 {
		return nil, errors.New("expected state M2")
	}
	if salt == nil && remotePubk == nil && m2err != 0x00 {
		return nil, errors.New("m2err = " + strconv.FormatInt(int64(m2err), 10))
	}

	clientSession, err := newPairSetupClientSession(salt, remotePubk, pin)
	if err != nil {
		return nil, err
	}

	return clientSession, nil
}

func (c *Controller) PairSetupM3(pairing *Pairing, clientSession *pairSetupClientSession) error {
	fmt.Println("PairSetupM3")

	ep := fmt.Sprintf("%s/%s", pairing.httpAddr, "pair-setup")

	// m3
	m3 := pairSetupM3Payload{
		State:     M3,
		PublicKey: clientSession.PublicKey,
		Proof:     clientSession.Proof,
	}
	b, err := tlv8.Marshal(m3)
	if err != nil {
		return err
	}

	resp, err := pairing.httpc.Post(ep, HTTPContentTypePairingTLV8, bytes.NewReader(b))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid status code %v", resp.StatusCode)
	}
	res := resp.Body
	defer res.Close()
	all, err := ioutil.ReadAll(res)
	if err != nil {
		return err
	}
	m4 := pairSetupPayload{}
	err = tlv8.Unmarshal(all, &m4)
	if err != nil {
		return err
	}

	state := m4.State
	if state != M4 {
		return errors.New("expected state M4")
	}
	serverProof := m4.Proof
	m4err := m4.Error
	if serverProof == nil && m4err != 0x00 {
		return errors.New("m4err = " + strconv.FormatInt(int64(m4err), 10))
	}
	serverProofValid := clientSession.session.VerifyServerAuthenticator(serverProof)
	if !serverProofValid {
		return errors.New("server proof is not valid")
	}

	return nil
}

func (c *Controller) PairSetupM5(pairing *Pairing, clientSession *pairSetupClientSession) error {
	fmt.Println("PairSetupM5")
	ep := fmt.Sprintf("%s/%s", pairing.httpAddr, "pair-setup")

	err := clientSession.SetupEncryptionKey(
		[]byte("Pair-Setup-Encrypt-Salt"),
		[]byte("Pair-Setup-Encrypt-Info"),
	)
	if err != nil {
		return err
	}

	hash, err := hkdf.Sha512(
		clientSession.SessionKey,
		[]byte("Pair-Setup-Controller-Sign-Salt"),
		[]byte("Pair-Setup-Controller-Sign-Info"),
	)
	if err != nil {
		return err
	}

	var material []byte
	material = append(material, hash[:]...)
	material = append(material, c.name...)
	material = append(material, c.localLTKP...)

	signature, err := ed25519.Signature(c.localLTSK, material)
	if err != nil {
		return err
	}

	m5raw := pairSetupM5RawPayload{
		Identifier: c.name,
		PublicKey:  c.localLTKP,
		Signature:  signature,
	}
	b, err := tlv8.Marshal(m5raw)
	if err != nil {
		return err
	}

	encryptedBytes, tag, err := chacha20poly1305.EncryptAndSeal(
		clientSession.EncryptionKey[:],
		[]byte("PS-Msg05"),
		b,
		nil,
	)
	if err != nil {
		return err
	}

	encData := append(encryptedBytes, tag[:]...)
	m5enc := pairSetupM5EncPayload{
		Method:        0,
		State:         M5,
		EncryptedData: encData,
	}
	b, err = tlv8.Marshal(m5enc)
	if err != nil {
		return err
	}
	resp, err := pairing.httpc.Post(ep, HTTPContentTypePairingTLV8, bytes.NewReader(b))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid status code %v", resp.StatusCode)
	}
	res := resp.Body
	defer res.Close()
	all, err := ioutil.ReadAll(res)
	if err != nil {
		return err
	}
	m6enc := pairSetupPayload{}
	err = tlv8.Unmarshal(all, &m6enc)
	if err != nil {
		return err
	}
	if m6enc.EncryptedData == nil && m6enc.Error != 0x00 {
		return errors.New("m6err = " + strconv.FormatInt(int64(m6enc.Error), 10))
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
		return err
	}

	//log.Println("m6dec.State = ", m6dec.State) somehow it's state is 0
	//if m6dec.State != M6 {
	//	return errors.New("expected state M6")
	//}
	if m6dec.PublicKey == nil && m6dec.Error != 0x00 {
		return errors.New("m6err = " + strconv.FormatInt(int64(m6dec.Error), 10))
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
		return err
	}

	accessoryInfo := hash[:]
	accessoryInfo = append(accessoryInfo, accessoryId...)
	accessoryInfo = append(accessoryInfo, accessoryLTPK...)

	valid := ed25519.ValidateSignature(accessoryLTPK, accessoryInfo, accessorySignature)
	if !valid {
		return errors.New("m6 sig not valid")
	}

	pairing.Name = accessoryId
	pairing.PublicKey = accessoryLTPK
	//pairing.tcpAddr = devTcpAddr
	pairing.discovered = true
	pairing.verified = false
	return nil
}

func (c *Controller) PairSetup(deviceId string, pin string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var ok bool
	var pairing *Pairing

	_, ok = c.mdnsDiscovered[deviceId]
	if !ok {
		return errors.New("device with given id not discovered")
	}
	if pairing, ok = c.pairings[deviceId]; !ok {
		fmt.Println("pairing not found")
		return errors.New("pairing not found")
	}

	if pairing.paired {
		fmt.Println("already paired!")
		return nil
	}

	clientSession, err := c.PairSetupM1(pairing, pin)
	if err != nil {
		return err
	}
	err = c.PairSetupM3(pairing, clientSession)
	if err != nil {
		return err
	}
	err = c.PairSetupM5(pairing, clientSession)
	if err != nil {
		return err
	}

	//c.pairings[accessoryId] = &pairing

	err = c.st.SavePairing(*pairing)
	if err != nil {
		return err
	}

	return nil
}