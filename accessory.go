package hkontroller

type Accessory struct {
	Id uint64                `json:"aid"`
	Ss []*ServiceDescription `json:"services"`
}

type Accessories struct {
	Accs []*Accessory `json:"accessories,omitempty"`
}

func (a *Accessory) GetService(serviceType HapServiceType) *ServiceDescription {

	for _, s := range a.Ss {
		if s.Type.ToShort() == serviceType.ToShort() {
			return s
		}
	}

	return nil
}
