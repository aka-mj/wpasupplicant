package main

import (
	"gitlab.com/michaeljohn/wpasupplicant"
)

func main() {
	var (
		uconn *wpasupplicant.Conn
		id    int
	)
	uconn, _ = wpasupplicant.Connect("/var/run/wpa_supplicant")
	defer uconn.Close()
	id, _ = uconn.AddNetwork()
	uconn.SetNetworkQuoted(id, "ssid", "foonet")
	uconn.SetNetworkQuoted(id, "psk", "pass")
	uconn.SetNetwork(id, "proto", "WPA2")
	uconn.SetNetwork(id, "key_mgmt", "WPA-PSK")
	uconn.SelectNetwork(id)
}
