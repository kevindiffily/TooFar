package kasa

// defined by kasa devices
type kasaDevice struct {
	System kasaSystem `json:"system"`
}

// defined by kasa devices
type kasaSystem struct {
	Sysinfo kasaSysinfo `json:"get_sysinfo"`
}

// defined by kasa devices
type kasaSysinfo struct {
	SWVersion  string `json:"sw_ver"`
	HWVersion  string `json:"hw_ver"`
	Model      string `json:"model"`
	DeviceID   string `json:"deviceId"`
	OEMID      string `json:"oemId"`
	HWID       string `json:"hwId"`
	RSSI       int    `json:"rssi"`
	Longitude  int    `json:"longitude_i"`
	Latitude   int    `json:"latitude_i"`
	Alias      string `json:"alias"`
	Status     string `json:"status"`
	MIC        string `json:"mic_type"`
	Feature    string `json:"feature"`
	MAC        string `json:"mac"`
	Updating   int    `json""updating"`
	LEDOff     int    `json:"led_off"`
	RelayState int    `json:"relay_state"`
	Brightness int    `json:"brightness"`
	OnTime     int    `json:"on_time"`
	ActiveMode string `json:"active_mode"`
	DevName    string `json:"dev_name"`
}
