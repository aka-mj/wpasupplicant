# Control Interface for wpa_supplicant in Go

[![ISC License](https://img.shields.io/badge/license-ISC-blue.svg)](https://gitlab.com/zfoo/wpasupplicant/blob/master/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/gitlab.com/zfoo/wpasupplicant)](https://goreportcard.com/report/gitlab.com/zfoo/wpasupplicant)

Package wpasupplicant provides a control interface to a wpa_supplicant process.

The connection to the wpa_supplicant process is by a Unix socket. The control
interface is defined in your wpa_supplicant.conf file.

Example of a wpa_supplicant.conf:

```
ctrl_interface=/var/run/wpa_supplicant
```

To open a connection:

```go
uconn, err := wpasupplicant.Connect("/tmp/our-socket", "/var/run/wpa_supplicant")
```

From this point you can start configuring for your network:

```go
uconn.SetNetworkQuoted(0, "ssid", "foo")
uconn.SetNetworkQuoted(0, "psk", "bar")
uconn.SetNetwork(0, "proto", "WPA2")
uconn.SetNetwork(0, "key_mgmt", "WPA-PSK")
```

For a complete example:

```go
package main

import (
	"gitlab.com/zfoo/wpasupplicant"
)

func main() {
	var (
		uconn *wpasupplicant.Conn
		id    int
	)
	uconn, _ = wpasupplicant.Connect("/tmp/our-socket", "/var/run/wpa_supplicant")
	defer uconn.Close()
	id, _ = uconn.AddNetwork()
	uconn.SetNetworkQuoted(id, "ssid", "foonet")
	uconn.SetNetworkQuoted(id, "psk", "pass")
	uconn.SetNetwork(id, "proto", "WPA2")
	uconn.SetNetwork(id, "key_mgmt", "WPA-PSK")
	uconn.SelectNetwork(id)
}

```

How to know when to use `SetNetwork` vs `SetNetworkQuoted`? Take a look at the wpa_supplicant.conf
documentation. If it needs to be enclosed in quotes in the configuration file then it needs to be
quoted here.

https://w1.fi/cgit/hostap/plain/wpa_supplicant/wpa_supplicant.conf

For further information on the wpa_supplicant control interface:

http://w1.fi/wpa_supplicant/devel/ctrl_iface_page.html
