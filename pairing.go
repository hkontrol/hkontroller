package hkontroller

type Pairing struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	PublicKey  []byte `json:"pubk"`
	Permission byte   `json:"permission,omitempty"`
}
