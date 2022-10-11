package hkontroller

/*
  {
    "iid": 2,
    "type": "14",
    "perms": [
      "pw"
    ],
    "format": "bool",
    "description": "Identify"
  },
	{
	  "iid": 11,
	  "type": "7C",
	  "perms": [
		"pr",
		"pw",
		"ev"
	  ],
	  "format": "uint8",
	  "value": 0,
	  "description": "Target Position",
	  "unit": "percentage",
	  "maxValue": 100,
	  "minValue": 0,
	  "minStep": 1
	},

*/

type CharacteristicDescription struct {
	Aid         uint64                `json:"aid"`
	Iid         uint64                `json:"iid"`
	Type        HapCharacteristicType `json:"type"`
	Value       interface{}           `json:"value"`
	Permissions []string              `json:"perms"`

	// optional values
	Format      *string     `json:"format,omitempty"`
	Status      *int        `json:"status,omitempty"`
	Events      *bool       `json:"ev,omitempty"`
	Unit        *string     `json:"unit,omitempty"`
	MinValue    interface{} `json:"minValue,omitempty"`
	MaxValue    interface{} `json:"maxValue,omitempty"`
	MinStep     interface{} `json:"minStep,omitempty"`
	MaxLen      *int        `json:"maxLen,omitempty"`
	ValidValues []int       `json:"valid-values,omitempty"`
	ValidRange  []int       `json:"valid-values-range,omitempty"`
}

type CharacteristicPut struct {
	Aid uint64 `json:"aid"`
	Iid uint64 `json:"iid"`

	Value  interface{} `json:"value,omitempty"`
	Status *int        `json:"status,omitempty"`
	Events *bool       `json:"ev,omitempty"`

	Remote   *bool `json:"remote,omitempty"`
	Response *bool `json:"r,omitempty"`
}
