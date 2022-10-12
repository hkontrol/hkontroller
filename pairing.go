package hkontroller

type Pairing struct {
	Name       string `json:"name"`
	PublicKey  []byte `json:"publicKey"`
	Permission byte   `json:"permission,omitempty"`
}
