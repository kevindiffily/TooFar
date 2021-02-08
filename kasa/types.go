package kasa

// defined by kasa devices
type kasaDevice struct {
	System    ksystem   `json:"system"`
	Countdown countdown `json:"count_down"`
	// emeter stuff?
}

// defined by kasa devices
type ksystem struct {
	Sysinfo ksysinfo `json:"get_sysinfo"`
}

// defined by kasa devices
type ksysinfo struct {
	SWVersion  string  `json:"sw_ver"`
	HWVersion  string  `json:"hw_ver"`
	Model      string  `json:"model"`
	DeviceID   string  `json:"deviceId"`
	OEMID      string  `json:"oemId"`
	HWID       string  `json:"hwId"`
	RSSI       int     `json:"rssi"`
	Longitude  int     `json:"longitude_i"`
	Latitude   int     `json:"latitude_i"`
	Alias      string  `json:"alias"`
	Status     string  `json:"status"`
	MIC        string  `json:"mic_type"`
	Feature    string  `json:"feature"`
	MAC        string  `json:"mac"`
	Updating   int     `json""updating"`
	LEDOff     int     `json:"led_off"`
	RelayState int     `json:"relay_state"`
	Brightness int     `json:"brightness"`
	OnTime     int     `json:"on_time"`
	ActiveMode string  `json:"active_mode"`
	DevName    string  `json:"dev_name"`
	Children   []child `json:"children"`
}

type child struct {
	ID         string `json:"id"`
	RelayState int    `json:"state"`
	Alias      string `json:"alias"`
	OnTime     int    `json:"on_time"`
}

type countdown struct {
	GetRules getRules `json:"get_rules"`
	DelRules delRules `json:"delete_all_rules"`
	AddRule  addRule  `json:"add_rule"`
}

type getRules struct {
	RuleList     []rule `json:"rule_list"`
	ErrorCode    int8   `json:"err_code"`
	ErrorMessage string `json:"err_msg"`
}

type rule struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Enable    uint8  `json:"enable"`
	Delay     uint16 `json:"delay"`
	Active    uint8  `json:"act"`
	Remaining uint16 `json:"remain"`
}

type delRules struct {
	ErrorCode    int8   `json:"err_code"`
	ErrorMessage string `json:"err_msg"`
}

type addRule struct {
	ID           string `json:"id"`
	ErrorCode    int8   `json:"err_code"`
	ErrorMessage string `json:"err_msg"`
}
