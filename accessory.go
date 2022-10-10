package hkontroller

type Accessory struct {
	Id uint64     `json:"aid"`
	Ss []*Service `json:"services"`
}

type Accessories struct {
	Accs []*Accessory `json:"accessories,omitempty"`
}

func (a *Accessory) GetAccessoryInfoService() *Service {
	for _, s := range a.Ss {
		if s.Type == AccessoryInfo {
			return s
		}
	}
	return nil
}
