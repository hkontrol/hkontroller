package hkontroller

type Pairing struct {
	Id         string `json:"id"`
	PublicKey  []byte `json:"pubk"`
	Permission byte   `json:"permission,omitempty"`
}
