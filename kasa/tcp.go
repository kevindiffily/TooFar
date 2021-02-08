package kasa

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"github.com/brutella/hc/log"
	tfaccessory "github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/config"
	"net"
	"time"
)

func getSettingsTCP(a *tfaccessory.TFAccessory) (*ksysinfo, error) {
	// log.Info.Printf("full kasa pull for [%s]", a.Name)
	res, err := sendTCP(a.IP, cmd_sysinfo)
	if err != nil {
		log.Info.Println(err.Error())
		return nil, err
	}
	// log.Info.Println(res)

	var kd kasaDevice
	if err = json.Unmarshal([]byte(res), &kd); err != nil {
		log.Info.Println(err.Error())
		return nil, err
	}
	// log.Info.Printf("%+v", kd.System.Sysinfo)
	return &kd.System.Sysinfo, nil
}

func encryptTCP(plaintext string) []byte {
	n := len(plaintext)
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, uint32(n))
	ciphertext := []byte(buf.Bytes())

	key := byte(0xAB)
	payload := make([]byte, n)
	for i := 0; i < n; i++ {
		payload[i] = plaintext[i] ^ key
		key = payload[i]
	}

	for i := 0; i < len(payload); i++ {
		ciphertext = append(ciphertext, payload[i])
	}

	return ciphertext
}

func sendTCP(ip string, cmd string) (string, error) {
	timeout := config.Get().KasaTimeout
	// unset/0/1 -- use the default of 10 seconds
	if timeout <= 0 {
		timeout = 10
	}
	payload := encryptTCP(cmd)
	r := net.TCPAddr{
		IP:   net.ParseIP(ip),
		Port: 9999,
	}

	conn, err := net.DialTCP("tcp4", nil, &r)
	if err != nil {
		log.Info.Printf("Cannot connnect to device: %s", err.Error())
		return "", err
	}
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(timeout)))
	_, err = conn.Write(payload)
	if err != nil {
		log.Info.Printf("Cannot send command to device: %s", err.Error())
		return "", err
	}

	// HS200's return ~600 bytes, HS220's return ~800 bytes; 1k should be enough
	// KP103s return larger packets, bump this to 1500 (normal wifi mtu)
	// if we need more, we need to build a buffer and fill it over multiple reads
	// see go-eiscp's method for how to improve this
	data := make([]byte, 1500)
	n, err := conn.Read(data)
	if err != nil {
		log.Info.Println("Cannot read data from device:", err)
		return "", err
	}
	result := decrypt(data[4:n]) // start reading at 4, go to total bytes read
	return result, nil
}
