## command line HomeKit controller

this exampe is CLI for Controller.

## accessories

One can use `homebridge` with `homebridge-dummy` plugin.

```text
sudo npm i -g homebridge
sudo npm i -g homebridge-dummy
sudo npm i -g homebridge-dummy-thermostat
sudo npm -g i homebridge-fake-light-bulb
```

My `~/.homebridge/config.json`:

```json
{
  "bridge": {
    "name": "homebridge",
    "username": "CC:22:3D:E3:CE:30",
    "manufacturer": "homebridge.io",
    "model": "homebridge",
    "port": 51827,
    "pin": "031-45-154"
  },

  "description": "",
  "ports": {
    "start": 52100,
    "end": 52150,
    "comment": "ports to listen."
  },
  "accessories": [
    {
      "accessory": "DummySwitch",
      "name": "switch_1"
    },
    {
      "accessory": "DummySwitch",
      "name": "switch_2"
    },
    {
      "accessory": "DummySwitch",
      "name": "switch_3",
      "time": 18000,
      "random": true
    },
    {
      "accessory": "Thermostat",
      "name": "Thermostat"
    },
    {
      "name": "Simple Light",
      "brightness": false,
      "color": "none",
      "accessory": "homebridge-fake-light-bulb"
    },
    {
      "name": "Dimmer",
      "brightness": true,
      "color": "none",
      "accessory": "homebridge-fake-light-bulb"
    },
    {
      "name": "RGB",
      "brightness": true,
      "color": "hue",
      "accessory": "homebridge-fake-light-bulb"
    },
    {
      "name": "Light w temp",
      "brightness": true,
      "color": "colorTemperature",
      "accessory": "homebridge-fake-light-bulb"
    }
  ]
}
```

## this example

```text
$ go run examples/example.go
> help
commands: help
          devices
          use <device>
if device selected:
          pair <pin>
          unpair
          accessories
          get <aid> <iid>
          put <aid> <iid> <type:number/bool> <value>
          watch <aid> <iid>
          unwatch <aid> <iid>
          quit

```

One should use `devices` command to get list of discovered or previously paired devices.

Let, for example, controlled device have id `CC:22:3D:E3:CE:30`. If it is online and advertising itself, running `devices` command produce output:

```text
> devices
ID	FriendlyName	DNSSD	Paired	Verified
CC:22:3D:E3:CE:30	homebridge\ CAD8	discovered	---	---
```

Then, put `use CC:22:3D:E3:CE:30` command to select that device.
After that - pair:

```text
> use CC:22:3D:E3:CE:30
selected device:  CC:22:3D:E3:CE:30 	 homebridge\ CAD8
CC:22:3D:E3:CE:30> pair 031-45-154
pair-setup error:  m2err = 6
```

In my case homebridge instance is already paired, so I do `rm -rf ~/.homebridge/persist` and restart homebridge. After that:

```text
CC:22:3D:E3:CE:30> pair 031-45-154
device paired
establishing encrypted session
CC:22:3D:E3:CE:30> should be connected now
> 
```

Example of commands:

```text
CC:22:3D:E3:CE:30> accessories

# 1 homebridge CAD8
    │
    ├─service:  AccessoryInfo
    │  ├─ characteristic # 2 	[ Identify ] =  <nil>
    │  ├─ characteristic # 3 	[ Manufacturer ] =  homebridge.io
    │  ├─ characteristic # 4 	[ Model ] =  homebridge
    │  ├─ characteristic # 5 	[ Name ] =  homebridge CAD8
    │  ├─ characteristic # 6 	[ SerialNumber ] =  CC:22:3D:E3:CE:30
    │  └─ characteristic # 7 	[ FirmwareRevision ] =  1.5.0
    │
    └─service:  HapProtocolInfo
       └─ characteristic # 9 	[ Version ] =  1.1.0

# 2 switch_1
    │
    ├─service:  AccessoryInfo
    │  ├─ characteristic # 2 	[ Identify ] =  <nil>
    │  ├─ characteristic # 3 	[ Manufacturer ] =  Homebridge
    │  ├─ characteristic # 4 	[ Model ] =  Dummy Switch
    │  ├─ characteristic # 5 	[ Name ] =  switch_1
    │  ├─ characteristic # 6 	[ SerialNumber ] =  Dummy-switch_1
    │  └─ characteristic # 7 	[ FirmwareRevision ] =  0.7.0
    │
    └─service:  Switch
       ├─ characteristic # 9 	[ Name ] =  switch_1
       └─ characteristic # 10 	[ On ] =  0
....
....
```

To turn on `switch_1` run `put` command with aid=2(accessory id) and iid=10(characteristic id):

```text
CC:22:3D:E3:CE:30> put 2 10 bool true
CC:22:3D:E3:CE:30> put 2 10 number 1
```

Note: keep in mind that homebridge-dummy automatically turn it off after short amount of time.

