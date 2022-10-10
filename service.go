package hkontrol

type Service struct {
	Id      uint64            `json:"iid"`
	Type    HapServiceType    `json:"type"`
	Cs      []*Characteristic `json:"characteristics"`
	Hidden  *bool             `json:"hidden,omitempty"`
	Primary *bool             `json:"primary,omitempty"`
	Linked  []uint64          `json:"linked,omitempty"`
}
