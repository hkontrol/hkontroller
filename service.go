package hkontroller

type ServiceDescription struct {
	Id      uint64                       `json:"iid"`
	Type    HapServiceType               `json:"type"`
	Cs      []*CharacteristicDescription `json:"characteristics"`
	Hidden  *bool                        `json:"hidden,omitempty"`
	Primary *bool                        `json:"primary,omitempty"`
	Linked  []uint64                     `json:"linked,omitempty"`
}

func (s *ServiceDescription) GetCharacteristic(characteristicType HapCharacteristicType) *CharacteristicDescription {

	for _, c := range s.Cs {
		if c.Type.ToShort() == characteristicType.ToShort() {
			return c
		}
	}

	return nil
}
