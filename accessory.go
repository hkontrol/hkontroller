package hkontroller

import (
	"sort"
	"strings"
)

type Accessory struct {
	Id uint64                `json:"aid"`
	Ss []*ServiceDescription `json:"services"`
}

type Accessories struct {
	Accs []*Accessory `json:"accessories,omitempty"`
}

func (a *Accessory) GetService(serviceType HapServiceType) *ServiceDescription {

	idx := sort.Search(len(a.Ss), func(i int) bool {
		return strings.Compare(string(a.Ss[i].Type), string(serviceType)) >= 0
	})

	if idx > -1 && idx < len(a.Ss) {
		return a.Ss[idx]
	}

	return nil
}
