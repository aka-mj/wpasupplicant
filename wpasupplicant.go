// See LICENSE file for copyright and license details.

/*
Package wpasupplicant provides a control interface to a wpa_supplicant process.

The connection to the wpa_supplicant process is by a Unix socket. The control
interface is defined in your wpa_supplicant.conf file.

Example of a wpa_supplicant.conf:

	ctrl_interface=/var/run/wpa_supplicant

To open a connection:

	uconn, err := wpasupplicant.Connect("/tmp/our-socket", "/var/run/wpa_supplicant")

From this point you can start configuring for your network:

	uconn.SetNetworkQuoted(id, "ssid", "foo")
	uconn.SetNetworkQuoted(id, "psk", "bar")
	uconn.SetNetwork(id, "proto", "WPA2")
	uconn.SetNetwork(id, "key_mgmt", "WPA-PSK")

How to know when to use SetNetwork vs SetNetworkQuoted? Read the wpa_supplicant.conf
documentation.

https://w1.fi/cgit/hostap/plain/wpa_supplicant/wpa_supplicant.conf

For further information on the wpa_supplicant control interface:

http://w1.fi/wpa_supplicant/devel/ctrl_iface_page.html
*/
package wpasupplicant

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// A KeyFormat represents the format a key is in.
type KeyFormat int

// Possible key formats.
const (
	KeyASCII KeyFormat = iota
	KeyHex   KeyFormat = iota
)

// A Conn is a Unix socket connection to the wpa_supplicant process.
type Conn struct {
	uconn     *net.UnixConn
	localSock string
}

func (c *Conn) ok() bool {
	return c != nil && c.uconn != nil
}

// Connect creates a Unix socket connection to the wpa_supplicant process.
// The rsock is the wpa_supplicant process socket.
func Connect(rsock string) (*Conn, error) {
	var (
		uc    *Conn
		err   error
		local *os.File
	)
	uc = &Conn{}

	local, err = ioutil.TempFile("/tmp", "wpa_supplicant")
	if err != nil {
		return nil, err
	}
	os.Remove(local.Name())

	uc.localSock = local.Name()
	uc.uconn, err = net.DialUnix("unixgram",
		&net.UnixAddr{Name: local.Name(), Net: "unixgram"},
		&net.UnixAddr{Name: rsock, Net: "unixgram"})

	return uc, err
}

// Close the socket connection.
func (c *Conn) Close() error {
	var err error
	if c.ok() {
		err = c.uconn.Close()
	}
	// We ignore the error returned here, we should have permissions to remove
	// the file and it should still exist. If the permissions changed there
	// is little we can do here. If the file was deleted by some external
	// process then we can just move on.
	os.Remove(c.localSock)
	return err
}

// Send a request to wpa_supplicant and return the reply.
func (c *Conn) sendRequest(msg string) (reply []byte, err error) {
	var n int

	if !c.ok() {
		return reply, syscall.EINVAL
	}

	if n, err = c.uconn.Write([]byte(msg)); err != nil || n != len(msg) {
		return reply, fmt.Errorf("Error sending request: %v", err)
	}
	reply = make([]byte, 4096)
	n, err = c.uconn.Read(reply)
	return reply[:n], err
}

// Check that a relpy message from wpa_supplicant was the 'OK' message.
func checkReplyOk(reply []byte) error {
	if !bytes.Equal([]byte("OK\n"), reply) {
		return fmt.Errorf("%v", string(reply))
	}
	return nil
}

// Sends a request message to wpa_supplicant and checks the reply was OK.
func (c *Conn) sendRequestOk(msg string) error {
	var (
		reply []byte
		err   error
	)

	if reply, err = c.sendRequest(msg); err != nil {
		return err
	}

	return checkReplyOk(reply)
}

// SetNetwork sends a 'SET_NETWORK' request to wpa_supplicant.
func (c *Conn) SetNetwork(id int, field, value string) error {
	return c.sendRequestOk(fmt.Sprintf("SET_NETWORK %v %v %v", id, field, value))
}

// SetNetworkQuoted sends a 'SET_NETWORK' request to wpa_supplicant and adds quotes
// around value.
func (c *Conn) SetNetworkQuoted(id int, field, value string) error {
	return c.SetNetwork(id, field, fmt.Sprintf("\"%v\"", value))
}

// SetNetworkWepKeys is a convenience method for setting the WEP keys.
// If the format is ASCII, the method adds the quotes around the keys
// automatically. So you can write "ABCD" rather than "\"ABCD\"".
func (c *Conn) SetNetworkWepKeys(id int, format KeyFormat, keys []string) error {
	for i, value := range keys {
		switch format {
		case KeyASCII:
			if err := c.SetNetworkQuoted(id, fmt.Sprintf("wep_key%v", i), value); err != nil {
				return err
			}
		case KeyHex:
			if err := c.SetNetwork(id, fmt.Sprintf("wep_key%v", i), value); err != nil {
				return err
			}
		}
	}
	return nil
}

// GetNetwork sends a 'GET_NETWORK' request to wpa_supplicant and returns
// the value of the given field.
func (c *Conn) GetNetwork(id int, field string) (value string, err error) {
	var (
		reply []byte
	)

	if reply, err = c.sendRequest(fmt.Sprintf("GET_NETWORK %v %v", id, field)); err != nil {
		return "", err
	}

	return string(reply), nil
}

// AddNetwork adds a new, empty network.
func (c *Conn) AddNetwork() (id int, err error) {
	var (
		reply []byte
	)

	if reply, err = c.sendRequest("ADD_NETWORK"); err != nil {
		return -1, err
	}
	return strconv.Atoi(strings.TrimSpace(string(reply)))
}

// RemoveNetwork removes a network.
func (c *Conn) RemoveNetwork(id int) error {
	return c.sendRequestOk(fmt.Sprintf("REMOVE_NETWORK %v", id))
}

// RemoveAllNetworks removes all networks.
func (c *Conn) RemoveAllNetworks() error {
	return c.sendRequestOk("REMOVE_NETWORK all")
}

// SetGlobalParameter sends a 'SET' request to wpa_supplicant.
func (c *Conn) SetGlobalParameter(field, value string) error {
	return c.sendRequestOk(fmt.Sprintf("SET %v %v", field, value))
}

// SelectNetwork selects the given network and disables the others.
func (c *Conn) SelectNetwork(id int) error {
	return c.sendRequestOk(fmt.Sprintf("SELECT_NETWORK %v", id))
}

// EnableNetwork enables a network.
func (c *Conn) EnableNetwork(id int) error {
	return c.sendRequestOk(fmt.Sprintf("ENABLE_NETWORK %v", id))
}

// EnableAllNetworks enables all networks.
func (c *Conn) EnableAllNetworks() error {
	return c.sendRequestOk("ENABLE_NETWORK all")
}

// DisableNetwork disables a network.
func (c *Conn) DisableNetwork(id int) error {
	return c.sendRequestOk(fmt.Sprintf("DISABLE_NETWORK %v", id))
}

// DisableAllNetworks disables all networks.
func (c *Conn) DisableAllNetworks() error {
	return c.sendRequestOk("DISABLE_NETWORK all")
}

// Reassociate forces a reassociation.
func (c *Conn) Reassociate() error {
	return c.sendRequestOk("REASSOCIATE")
}

// Reconnect will attempt to connect if in a disconnected state.
func (c *Conn) Reconnect() error {
	return c.sendRequestOk("RECONNECT")
}

// ListNetworks returns a list of configured networks.
func (c *Conn) ListNetworks() (string, error) {
	var (
		reply []byte
		err   error
	)

	reply, err = c.sendRequest("LIST_NETWORKS")
	return string(reply), err
}

// NumOfNetworks returns the number of networks configured.
func (c *Conn) NumOfNetworks() (int, error) {
	var (
		reply string
		err   error
	)

	if reply, err = c.ListNetworks(); err != nil {
		return 0, err
	}

	// we don't want to include the header in the count
	return strings.Count(reply, "\n") - 1, nil
}

// Reconfigure forces wpa_supplicant to re-read its configuration data.
// This will wipe out any networks configured at run time.
func (c *Conn) Reconfigure() error {
	return c.sendRequestOk("RECONFIGURE")
}

// Status information for current WPA/EAPOL/EAP connection.
func (c *Conn) Status() (string, error) {
	var (
		reply []byte
		err   error
	)

	reply, err = c.sendRequest("STATUS")
	return string(reply), err
}

// StatusVerbose returns more information than Status() (on some machines).
func (c *Conn) StatusVerbose() (string, error) {
	var (
		reply []byte
		err   error
	)

	reply, err = c.sendRequest("STATUS-VERBOSE")
	return string(reply), err
}

// Ping is used to test whether wpa_supplicant is replying to the control
// interface command. It returns nil if the connection is open and
// processing commands.
func (c *Conn) Ping() error {
	var (
		reply []byte
		err   error
	)

	if reply, err = c.sendRequest("PING"); err != nil {
		return err
	}
	if !bytes.Equal([]byte("PONG\n"), reply) {
		return fmt.Errorf("Received unexpected reply: %v", string(reply))
	}
	return nil
}

// SaveConfig saves the current configuration.
func (c *Conn) SaveConfig() error {
	return c.sendRequestOk("SAVE_CONFIG")
}

// Interfaces list available interfaces.
func (c *Conn) Interfaces() (string, error) {
	var (
		reply []byte
		err   error
	)

	reply, err = c.sendRequest("INTERFACES")
	return string(reply), err
}

// Ifname returns the current interface name.
func (c *Conn) Ifname() (string, error) {
	var (
		reply []byte
		err   error
	)

	reply, err = c.sendRequest("IFNAME")
	return string(reply), err
}

// BSS returns detailed per-BSS scan results.
func (c *Conn) BSS(id int) (string, error) {
	var (
		reply []byte
		err   error
	)

	reply, err = c.sendRequest(fmt.Sprintf("BSS %v", id))
	return string(reply), err
}
