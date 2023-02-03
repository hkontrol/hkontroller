package hkontroller

type Pairing struct {
	Name       string `json:"name"`
	Id         string `json:"id"`
	PublicKey  []byte `json:"pubk"`
	Permission byte   `json:"permission,omitempty"`
}
