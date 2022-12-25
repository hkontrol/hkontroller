package hkontroller

import "strings"

type HapServiceType string

const (
	SType_HapProtocolInfo               HapServiceType = "A2"
	SType_AccessoryInfo                 HapServiceType = "3E"
	SType_AirPurifier                   HapServiceType = "BB"
	SType_AirQualitySensor              HapServiceType = "8D"
	SType_AudioStreamManagement         HapServiceType = "127"
	SType_BatteryService                HapServiceType = "96"
	SType_CameraRTPStreamManagement     HapServiceType = "110"
	SType_CarbonDioxideSensor           HapServiceType = "97"
	SType_CarbonMonoxideSensor          HapServiceType = "7F"
	SType_ContactSensor                 HapServiceType = "80"
	SType_DataStreamTransportManagement HapServiceType = "129"
	SType_Door                          HapServiceType = "81"
	SType_Doorbell                      HapServiceType = "121"
	SType_Fan                           HapServiceType = "B7"
	SType_Faucet                        HapServiceType = "D7"
	SType_FilterMaintenance             HapServiceType = "BA"
	SType_GarageDoorOpener              HapServiceType = "41"
	SType_HeaterCooler                  HapServiceType = "BC"
	SType_HumidifierDehumidifier        HapServiceType = "BD"
	SType_HumiditySensor                HapServiceType = "82"
	SType_IrrigationSystem              HapServiceType = "CF"
	SType_LeakSensor                    HapServiceType = "83"
	SType_LightBulb                     HapServiceType = "43"
	SType_LightSensor                   HapServiceType = "84"
	SType_LockManagement                HapServiceType = "44"
	SType_LockMechanism                 HapServiceType = "45"
	SType_Microphone                    HapServiceType = "112"
	SType_MotionSensor                  HapServiceType = "85"
	SType_OccupancySensor               HapServiceType = "86"
	SType_Outlet                        HapServiceType = "47"
	SType_SecuritySystem                HapServiceType = "7E"
	SType_ServiceLabel                  HapServiceType = "CC"
	SType_Siri                          HapServiceType = "133"
	SType_Slat                          HapServiceType = "B9"
	SType_SmokeSensor                   HapServiceType = "87"
	SType_Speaker                       HapServiceType = "113"
	SType_StatelessProgrammableSwitch   HapServiceType = "89"
	SType_Switch                        HapServiceType = "49"
	SType_TargetControl                 HapServiceType = "125"
	SType_TargetControlManagement       HapServiceType = "122"
	SType_TemperatureSensor             HapServiceType = "8A"
	SType_Thermostat                    HapServiceType = "4A"
	SType_Valve                         HapServiceType = "D0"
	SType_Window                        HapServiceType = "8B"
	SType_WindowCovering                HapServiceType = "8C"
)

func (h HapServiceType) String() string {
	switch h {
	case SType_AccessoryInfo:
		return "AccessoryInfo"
	case SType_HapProtocolInfo:
		return "HapProtocolInfo"
	case SType_LightBulb:
		return "LightBulb"
	case SType_AirPurifier:
		return "AirPurifier"
	case SType_AirQualitySensor:
		return "AirQualitySensor"
	case SType_AudioStreamManagement:
		return "AudioStreamManagement"
	case SType_BatteryService:
		return "BatteryService"
	case SType_CameraRTPStreamManagement:
		return "CameraRTPStreamManagement"
	case SType_CarbonDioxideSensor:
		return "CarbonDioxideSensor"
	case SType_CarbonMonoxideSensor:
		return "CarbonMonoxideSensor"
	case SType_ContactSensor:
		return "ContactSensor"
	case SType_DataStreamTransportManagement:
		return "DataStreamTransportManagement"
	case SType_Door:
		return "Door"
	case SType_Doorbell:
		return "Doorbell"
	case SType_Fan:
		return "Fan"
	case SType_Faucet:
		return "Faucet"
	case SType_FilterMaintenance:
		return "FilterMaintenance"
	case SType_GarageDoorOpener:
		return "GarageDoorOpener"
	case SType_HeaterCooler:
		return "HeaterCooler"
	case SType_HumidifierDehumidifier:
		return "HumidifierDehumidifier"
	case SType_HumiditySensor:
		return "HumiditySensor"
	case SType_IrrigationSystem:
		return "IrrigationSystem"
	case SType_LeakSensor:
		return "LeakSensor"
	case SType_LightSensor:
		return "LightSensor"
	case SType_LockManagement:
		return "LockManagement"
	case SType_LockMechanism:
		return "LockMechanism"
	case SType_Microphone:
		return "Microphone"
	case SType_MotionSensor:
		return "MotionSensor"
	case SType_OccupancySensor:
		return "OccupanceSensor"
	case SType_Outlet:
		return "Outlet"
	case SType_SecuritySystem:
		return "SecuritySystem"
	case SType_ServiceLabel:
		return "ServiceLabel"
	case SType_Siri:
		return "Siri"
	case SType_Slat:
		return "Slat"
	case SType_SmokeSensor:
		return "SmokeSensor"
	case SType_Speaker:
		return "Speaker"
	case SType_StatelessProgrammableSwitch:
		return "StatelessProgrammableSwitch"
	case SType_Switch:
		return "Switch"
	case SType_TargetControl:
		return "TargetControl"
	case SType_TargetControlManagement:
		return "TargetControlManagement"
	case SType_TemperatureSensor:
		return "TemperatureSensor"
	case SType_Thermostat:
		return "Thermostat"
	case SType_Valve:
		return "Valve"
	case SType_Window:
		return "Window"
	case SType_WindowCovering:
		return "WindowCovering"
	}

	return string(h)
}

func (h HapServiceType) ToShort() HapServiceType {
	s := string(h)
	a := strings.Split(s, "-")
	s = a[0]
	for i := 0; i < len(s); i += 1 {
		if s[i] != '0' {
			s = s[i:]
			return HapServiceType(s)
		}
	}
	return h
}

type HapCharacteristicType string

const (
	CType_Identify                                  HapCharacteristicType = "14"
	CType_Manufacturer                              HapCharacteristicType = "20"
	CType_Model                                     HapCharacteristicType = "21"
	CType_Name                                      HapCharacteristicType = "23"
	CType_SerialNumber                              HapCharacteristicType = "30"
	CType_Version                                   HapCharacteristicType = "37"
	CType_FirmwareRevision                          HapCharacteristicType = "52"
	CType_HardwareRevision                          HapCharacteristicType = "53"
	CType_On                                        HapCharacteristicType = "25"
	CType_Brightness                                HapCharacteristicType = "8"
	CType_AccessoryFlags                            HapCharacteristicType = "A6"
	CType_Active                                    HapCharacteristicType = "B0"
	CType_ActiveIdentifier                          HapCharacteristicType = "E7"
	CType_AdministratorOnlyAccess                   HapCharacteristicType = "1"
	CType_AudioFeedback                             HapCharacteristicType = "5"
	CType_AirParticulateSize                        HapCharacteristicType = "65"
	CType_AirQuality                                HapCharacteristicType = "95"
	CType_BatteryLevel                              HapCharacteristicType = "68"
	CType_ButtonEvent                               HapCharacteristicType = "126"
	CType_CarbonMonoxideLevel                       HapCharacteristicType = "90"
	CType_CarbonMonoxidePeakLevel                   HapCharacteristicType = "91"
	CType_CarbonMonoxideDetected                    HapCharacteristicType = "69"
	CType_CarbonDioxideLevel                        HapCharacteristicType = "93"
	CType_CarbonDioxidePeakLevel                    HapCharacteristicType = "94"
	CType_CarbonDioxideDetected                     HapCharacteristicType = "92"
	CType_ChargingState                             HapCharacteristicType = "8F"
	CType_CoolingThresholdTemperature               HapCharacteristicType = "D"
	CType_ColorTemperature                          HapCharacteristicType = "CE"
	CType_ContactSensorState                        HapCharacteristicType = "6A"
	CType_CurrentAmbientLightLevel                  HapCharacteristicType = "6B"
	CType_CurrentHorizontalTiltAngle                HapCharacteristicType = "6C"
	CType_CurrentAirPurifierState                   HapCharacteristicType = "A9"
	CType_CurrentSlatState                          HapCharacteristicType = "AA"
	CType_CurrentPosition                           HapCharacteristicType = "6D"
	CType_CurrentVerticalTiltAngle                  HapCharacteristicType = "6E"
	CType_CurrentHumidifierDehumidifierState        HapCharacteristicType = "B3"
	CType_CurrentDoorState                          HapCharacteristicType = "E"
	CType_CurrentFanState                           HapCharacteristicType = "AF"
	CType_CurrentHeatingCoolingState                HapCharacteristicType = "F"
	CType_CurrentHeaterCoolerState                  HapCharacteristicType = "B1"
	CType_CurrentRelativeHumidity                   HapCharacteristicType = "10"
	CType_CurrentTemperature                        HapCharacteristicType = "11"
	CType_CurrentTiltAngle                          HapCharacteristicType = "C1"
	CType_DigitalZoom                               HapCharacteristicType = "11D"
	CType_FilterLifeLevel                           HapCharacteristicType = "AB"
	CType_FilterChangeIndication                    HapCharacteristicType = "AC"
	CType_HeatingThresholdTemperature               HapCharacteristicType = "12"
	CType_HoldPosition                              HapCharacteristicType = "6F"
	CType_Hue                                       HapCharacteristicType = "13"
	CType_ImageRotation                             HapCharacteristicType = "11E"
	CType_ImageMirroring                            HapCharacteristicType = "11F"
	CType_InUse                                     HapCharacteristicType = "D2"
	CType_IsConfigured                              HapCharacteristicType = "D6"
	CType_LeakDetected                              HapCharacteristicType = "70"
	CType_LockControlPoint                          HapCharacteristicType = "19"
	CType_LockCurrentState                          HapCharacteristicType = "1D"
	CType_LockLastKnownAction                       HapCharacteristicType = "1C"
	CType_LockManagementAutoSecurityTimeout         HapCharacteristicType = "1A"
	CType_LockPhysicalControls                      HapCharacteristicType = "A7"
	CType_LockTargetState                           HapCharacteristicType = "1E"
	CType_Logs                                      HapCharacteristicType = "1F"
	CType_MotionDetected                            HapCharacteristicType = "22"
	CType_Mute                                      HapCharacteristicType = "11A"
	CType_NightVision                               HapCharacteristicType = "11B"
	CType_NitrogenDioxideDensity                    HapCharacteristicType = "C4"
	CType_ObstructionDetected                       HapCharacteristicType = "24"
	CType_PM25Density                               HapCharacteristicType = "C6"
	CType_OccupancyDetected                         HapCharacteristicType = "71"
	CType_OpticalZoom                               HapCharacteristicType = "11C"
	CType_OutletInUse                               HapCharacteristicType = "26"
	CType_OzoneDensity                              HapCharacteristicType = "C3"
	CType_PM10Density                               HapCharacteristicType = "C7"
	CType_PositionState                             HapCharacteristicType = "72"
	CType_ProgramMode                               HapCharacteristicType = "D1"
	CType_ProgrammableSwitchEvent                   HapCharacteristicType = "73"
	CType_RelativeHumidityDehumidifierThreshold     HapCharacteristicType = "C9"
	CType_RelativeHumidityHumidifierThreshold       HapCharacteristicType = "CA"
	CType_RemainingDuration                         HapCharacteristicType = "D4"
	CType_ResetFilterIndication                     HapCharacteristicType = "AD"
	CType_RotationDirection                         HapCharacteristicType = "28"
	CType_RotationSpeed                             HapCharacteristicType = "29"
	CType_Saturation                                HapCharacteristicType = "2F"
	CType_SecuritySystemAlarmType                   HapCharacteristicType = "BE"
	CType_SecuritySystemCurrentState                HapCharacteristicType = "66"
	CType_SecuritySystemTargetState                 HapCharacteristicType = "67"
	CType_SelectedAudioStreamConfiguration          HapCharacteristicType = "128"
	CType_ServiceLabelIndex                         HapCharacteristicType = "CB"
	CType_ServiceLabelNamespace                     HapCharacteristicType = "CD"
	CType_SetupDataStreamTransport                  HapCharacteristicType = "131"
	CType_SelectedRTPStreamConfiguration            HapCharacteristicType = "117"
	CType_SetupEndpoints                            HapCharacteristicType = "118"
	CType_SiriInputType                             HapCharacteristicType = "132"
	CType_SlatType                                  HapCharacteristicType = "C0"
	CType_SmokeDetected                             HapCharacteristicType = "76"
	CType_StatusActive                              HapCharacteristicType = "75"
	CType_StatusFault                               HapCharacteristicType = "77"
	CType_StatusJammed                              HapCharacteristicType = "78"
	CType_StatusLowBattery                          HapCharacteristicType = "79"
	CType_StatusTampered                            HapCharacteristicType = "7A"
	CType_StreamingStatus                           HapCharacteristicType = "120"
	CType_SupportedAudioStreamConfiguration         HapCharacteristicType = "115"
	CType_SupportedDataStreamTransportConfiguration HapCharacteristicType = "130"
	CType_SupportedRTPConfiguration                 HapCharacteristicType = "116"
	CType_SupportedVideoStreamConfiguration         HapCharacteristicType = "114"
	CType_SulphurDioxideDensity                     HapCharacteristicType = "C5"
	CType_SwingMode                                 HapCharacteristicType = "B6"
	CType_TargetAirPurifierState                    HapCharacteristicType = "A8"
	CType_TargetFanState                            HapCharacteristicType = "BF"
	CType_TargetTiltAngle                           HapCharacteristicType = "C2"
	CType_TargetHeaterCoolerState                   HapCharacteristicType = "B2"
	CType_SetDuration                               HapCharacteristicType = "D3"
	CType_TargetControlSupportedConfiguration       HapCharacteristicType = "123"
	CType_TargetControlList                         HapCharacteristicType = "124"
	CType_TargetHorizontalTiltAngle                 HapCharacteristicType = "7B"
	CType_TargetHumidifierDehumidifierState         HapCharacteristicType = "B4"
	CType_TargetPosition                            HapCharacteristicType = "7C"
	CType_TargetDoorState                           HapCharacteristicType = "32"
	CType_TargetHeatingCoolingState                 HapCharacteristicType = "33"
	CType_TargetRelativeHumidity                    HapCharacteristicType = "34"
	CType_TargetTemperature                         HapCharacteristicType = "35"
	CType_TemperatureDisplayUnits                   HapCharacteristicType = "36"
	CType_TargetVerticalTiltAngle                   HapCharacteristicType = "7D"
	CType_ValveType                                 HapCharacteristicType = "D5"
	CType_VOCDensity                                HapCharacteristicType = "C8"
	CType_Volume                                    HapCharacteristicType = "119"
	CType_WaterLevel                                HapCharacteristicType = "B5"
)

func (h HapCharacteristicType) String() string {
	switch h {
	case CType_Identify:
		return "Identify"
	case CType_Manufacturer:
		return "Manufacturer"
	case CType_Model:
		return "Model"
	case CType_Name:
		return "Name"
	case CType_SerialNumber:
		return "SerialNumber"
	case CType_Version:
		return "Version"
	case CType_FirmwareRevision:
		return "FirmwareRevision"
	case CType_HardwareRevision:
		return "HardwareRevision"
	case CType_On:
		return "On"
	case CType_Brightness:
		return "Brightness"
	case CType_AccessoryFlags:
		return "AccessoryFlags"
	case CType_Active:
		return "Active"
	case CType_ActiveIdentifier:
		return "ActiveIdentifier"
	case CType_AdministratorOnlyAccess:
		return "AdministratorOnlyAccess"
	case CType_AudioFeedback:
		return "AudioFeedback"
	case CType_AirParticulateSize:
		return "AirParticulateSize"
	case CType_AirQuality:
		return "AirQuality"
	case CType_BatteryLevel:
		return "BatteryLevel"
	case CType_ButtonEvent:
		return "ButtonEvent "
	case CType_CarbonMonoxideLevel:
		return "CarbonMonoxideLevel"
	case CType_CarbonMonoxidePeakLevel:
		return "CarbonMonoxidePeakLevel"
	case CType_CarbonMonoxideDetected:
		return "CarbonMonoxideDetected"
	case CType_CarbonDioxideLevel:
		return "CarbonDioxideLevel"
	case CType_CarbonDioxidePeakLevel:
		return "CarbonDioxidePeakLevel"
	case CType_CarbonDioxideDetected:
		return "CarbonDioxideDetected"
	case CType_ChargingState:
		return "ChargingState"
	case CType_CoolingThresholdTemperature:
		return "CoolingThresholdTemperature"
	case CType_ColorTemperature:
		return "ColorTemperature"
	case CType_ContactSensorState:
		return "ContactSensorState"
	case CType_CurrentAmbientLightLevel:
		return "CurrentAmbientLightLevel"
	case CType_CurrentHorizontalTiltAngle:
		return "CurrentHorizontalTiltAngle"
	case CType_CurrentAirPurifierState:
		return "CurrentAirPurifierState"
	case CType_CurrentSlatState:
		return "CurrentSlatState"
	case CType_CurrentPosition:
		return "CurrentPosition"
	case CType_CurrentVerticalTiltAngle:
		return "CurrentVerticalTiltAngle"
	case CType_CurrentHumidifierDehumidifierState:
		return "CurrentHumidifierDehumidifierState"
	case CType_CurrentDoorState:
		return "CurrentDoorState"
	case CType_CurrentFanState:
		return "CurrentFanState"
	case CType_CurrentHeatingCoolingState:
		return "CurrentHeatingCoolingState"
	case CType_CurrentHeaterCoolerState:
		return "CurrentHeaterCoolerState"
	case CType_CurrentRelativeHumidity:
		return "CurrentRelativeHumidity"
	case CType_CurrentTemperature:
		return "CurrentTemperature"
	case CType_CurrentTiltAngle:
		return "CurrentTiltAngle"
	case CType_DigitalZoom:
		return "DigitalZoom "
	case CType_FilterLifeLevel:
		return "FilterLifeLevel"
	case CType_FilterChangeIndication:
		return "FilterChangeIndication"
	case CType_HeatingThresholdTemperature:
		return "HeatingThresholdTemperature"
	case CType_HoldPosition:
		return "HoldPosition"
	case CType_Hue:
		return "Hue"
	case CType_ImageRotation:
		return "ImageRotation "
	case CType_ImageMirroring:
		return "ImageMirroring"
	case CType_InUse:
		return "InUse"
	case CType_IsConfigured:
		return "IsConfigured"
	case CType_LeakDetected:
		return "LeakDetected"
	case CType_LockControlPoint:
		return "LockControlPoint"
	case CType_LockCurrentState:
		return "LockCurrentState"
	case CType_LockLastKnownAction:
		return "LockLastKnownAction"
	case CType_LockManagementAutoSecurityTimeout:
		return "LockManagementAutoSecurityTimeout"
	case CType_LockPhysicalControls:
		return "LockPhysicalControls"
	case CType_LockTargetState:
		return "LockTargetState"
	case CType_Logs:
		return "Logs"
	case CType_MotionDetected:
		return "MotionDetected"
	case CType_Mute:
		return "Mute"
	case CType_NightVision:
		return "NightVision "
	case CType_NitrogenDioxideDensity:
		return "NitrogenDioxideDensity"
	case CType_ObstructionDetected:
		return "ObstructionDetected"
	case CType_PM25Density:
		return "PM25Density"
	case CType_OccupancyDetected:
		return "OccupancyDetected"
	case CType_OpticalZoom:
		return "OpticalZoom "
	case CType_OutletInUse:
		return "OutletInUse"
	case CType_OzoneDensity:
		return "OzoneDensity"
	case CType_PM10Density:
		return "PM10Density"
	case CType_PositionState:
		return "PositionState"
	case CType_ProgramMode:
		return "ProgramMode"
	case CType_ProgrammableSwitchEvent:
		return "ProgrammableSwitchEvent"
	case CType_RelativeHumidityDehumidifierThreshold:
		return "RelativeHumidityDehumidifierThreshold"
	case CType_RelativeHumidityHumidifierThreshold:
		return "RelativeHumidityHumidifierThreshold"
	case CType_RemainingDuration:
		return "RemainingDuration"
	case CType_ResetFilterIndication:
		return "ResetFilterIndication"
	case CType_RotationDirection:
		return "RotationDirection"
	case CType_RotationSpeed:
		return "RotationSpeed"
	case CType_Saturation:
		return "Saturation"
	case CType_SecuritySystemAlarmType:
		return "SecuritySystemAlarmType"
	case CType_SecuritySystemCurrentState:
		return "SecuritySystemCurrentState"
	case CType_SecuritySystemTargetState:
		return "SecuritySystemTargetState"
	case CType_SelectedAudioStreamConfiguration:
		return "SelectedAudioStreamConfiguration"
	case CType_ServiceLabelIndex:
		return "ServiceLabelIndex"
	case CType_ServiceLabelNamespace:
		return "ServiceLabelNamespace"
	case CType_SetupDataStreamTransport:
		return "SetupDataStreamTransport"
	case CType_SelectedRTPStreamConfiguration:
		return "SelectedRTPStreamConfiguration"
	case CType_SetupEndpoints:
		return "SetupEndpoints"
	case CType_SiriInputType:
		return "SiriInputType "
	case CType_SlatType:
		return "SlatType"
	case CType_SmokeDetected:
		return "SmokeDetected"
	case CType_StatusActive:
		return "StatusActive"
	case CType_StatusFault:
		return "StatusFault"
	case CType_StatusJammed:
		return "StatusJammed"
	case CType_StatusLowBattery:
		return "StatusLowBattery"
	case CType_StatusTampered:
		return "StatusTampered"
	case CType_StreamingStatus:
		return "StreamingStatus "
	case CType_SupportedAudioStreamConfiguration:
		return "SupportedAudioStreamConfiguration "
	case CType_SupportedDataStreamTransportConfiguration:
		return "SupportedDataStreamTransportConfiguration "
	case CType_SupportedRTPConfiguration:
		return "SupportedRTPConfiguration "
	case CType_SupportedVideoStreamConfiguration:
		return "SupportedVideoStreamConfiguration "
	case CType_SulphurDioxideDensity:
		return "SulphurDioxideDensity"
	case CType_SwingMode:
		return "SwingMode"
	case CType_TargetAirPurifierState:
		return "TargetAirPurifierState"
	case CType_TargetFanState:
		return "TargetFanState"
	case CType_TargetTiltAngle:
		return "TargetTiltAngle"
	case CType_TargetHeaterCoolerState:
		return "TargetHeaterCoolerState"
	case CType_SetDuration:
		return "SetDuration"
	case CType_TargetControlSupportedConfiguration:
		return "TargetControlSupportedConfiguration "
	case CType_TargetControlList:
		return "TargetControlList "
	case CType_TargetHorizontalTiltAngle:
		return "TargetHorizontalTiltAngle"
	case CType_TargetHumidifierDehumidifierState:
		return "TargetHumidifierDehumidifierState"
	case CType_TargetPosition:
		return "TargetPosition"
	case CType_TargetDoorState:
		return "TargetDoorState"
	case CType_TargetHeatingCoolingState:
		return "TargetHeatingCoolingState"
	case CType_TargetRelativeHumidity:
		return "TargetRelativeHumidity"
	case CType_TargetTemperature:
		return "TargetTemperature"
	case CType_TemperatureDisplayUnits:
		return "TemperatureDisplayUnits"
	case CType_TargetVerticalTiltAngle:
		return "TargetVerticalTiltAngle"
	case CType_ValveType:
		return "ValveType"
	case CType_VOCDensity:
		return "VOCDensity"
	case CType_Volume:
		return "Volume"
	case CType_WaterLevel:
		return "WaterLevel"
	}
	return string(h)
}

func (h HapCharacteristicType) ToShort() HapCharacteristicType {
	s := string(h)
	a := strings.Split(s, "-")
	s = a[0]
	for i := 0; i < len(s); i += 1 {
		if s[i] != '0' {
			s = s[i:]
			return HapCharacteristicType(s)
		}
	}
	return h
}
