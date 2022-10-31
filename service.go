package hkontroller

import (
	"sort"
	"strings"
)

type ServiceDescription struct {
	Id      uint64                       `json:"iid"`
	Type    HapServiceType               `json:"type"`
	Cs      []*CharacteristicDescription `json:"characteristics"`
	Hidden  *bool                        `json:"hidden,omitempty"`
	Primary *bool                        `json:"primary,omitempty"`
	Linked  []uint64                     `json:"linked,omitempty"`
}

func (s *ServiceDescription) GetCharacteristic(characteristicType HapCharacteristicType) *CharacteristicDescription {

	idx := sort.Search(len(s.Cs), func(i int) bool {
		return strings.Compare(string(s.Cs[i].Type), string(characteristicType)) >= 0
	})

	if idx > -1 && idx < len(s.Cs) {
		if s.Cs[idx].Type == characteristicType {
			return s.Cs[idx]
		} else {
			return nil
		}
	}

	return nil
}
