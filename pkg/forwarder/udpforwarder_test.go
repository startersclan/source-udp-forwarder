package udpforwarder

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

func TestForward(t *testing.T) {
	log.SetLevel(log.InfoLevel)
	listenAddr := "127.0.0.1:26999"
	forwardAddr := "127.0.0.1:27500"
	prependStr := "foo"
	forwarder, err := Forward(listenAddr, forwardAddr, DefaultTimeout, prependStr)
	if forwarder == nil || err != nil {
		t.Fatal(err)
	}
	defer forwarder.Close()
	time.Sleep(time.Millisecond * 100) // Don't stop too fast or else "bind: address already in use" in the next test
}

func TestHandleConnection(t *testing.T) {
	log.SetLevel(log.InfoLevel)
	listenAddr := "127.0.0.1:26999"
	forwardAddr := "127.0.0.1:27500"
	prependStr := "foo"
	forwarder, err := Forward(listenAddr, forwardAddr, DefaultTimeout, prependStr)
	if forwarder == nil || err != nil {
		t.Fatal(err)
	}
	defer forwarder.Close()

	log := "L 10/11/2019 - 23:41:02: Started map \"awp_city\" (CRC \"-2134348459\")"

	// Gameserver sends log to forwarder
	d := net.Dialer{
		LocalAddr: &net.UDPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 27000,
		},
	}
	conn, err := d.Dial("udp", listenAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	fmt.Fprintf(conn, log)
	fmt.Fprintf(conn, log)

	// Allow the time for the connection to be added
	time.Sleep(time.Millisecond * 100)

	connected := forwarder.Connected()
	if len(connected) == 0 {
		t.Fatalf("New connection not added!")
	}
}

func TestJanitor(t *testing.T) {
	// Setup forwarder
	log.SetLevel(log.InfoLevel)
	listenAddr := "127.0.0.1:26999"
	forwardAddr := "127.0.0.1:27500"
	timeout := time.Millisecond * 1
	prependStr := "foo"
	forwarder, err := Forward(listenAddr, forwardAddr, timeout, prependStr)
	if forwarder == nil || err != nil {
		t.Fatal(err)
	}
	defer forwarder.Close()

	// Gameserver sends log to forwarder
	log := "L 10/11/2019 - 23:41:02: Started map \"awp_city\" (CRC \"-2134348459\")"
	go func(tb testing.TB) {
		conn, err := net.Dial("udp", listenAddr)
		if err != nil {
			tb.Fatal(err)
		}
		defer conn.Close()
		fmt.Fprintf(conn, log)
	}(t)

	// Allow the janitor some time to cleanup
	time.Sleep(timeout * 100)

	connected := forwarder.Connected()
	if len(connected) > 0 {
		t.Fatalf("Stale connection not cleaned up: %s", connected[0])
	}
}

func TestForwardedUdp(t *testing.T) {
	// Setup forwarder
	log.SetLevel(log.InfoLevel)
	listenAddr := "127.0.0.1:26999"
	forwardAddr := "127.0.0.1:27500"
	prependStr := "foo"
	forwarder, err := Forward(listenAddr, forwardAddr, DefaultTimeout, prependStr)
	if forwarder == nil || err != nil {
		t.Fatal(err)
	}
	defer forwarder.Close()

	// Gameserver sends 3 log lines to forwarder
	log := "L 10/11/2019 - 23:41:02: Started map \"awp_city\" (CRC \"-2134348459\")"
	logs := []string{
		log,
		log,
		log,
	}
	go func(tb testing.TB) {
		time.Sleep(time.Millisecond * 300) // Wait for daemon to be up
		d := net.Dialer{
			LocalAddr: &net.UDPAddr{
				IP:   net.ParseIP("127.0.0.1"),
				Port: 27000,
			},
		}
		conn, err := d.Dial("udp", listenAddr)
		if err != nil {
			tb.Fatal(err)
		}
		defer conn.Close()
		for _, l := range logs {
			fmt.Fprintf(conn, l)
		}
	}(t)

	// Daemon receives log from forwarder
	expectedLog := prependStr + log
	addr, err := net.ResolveUDPAddr("udp", forwardAddr)
	connD, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer connD.Close()
	buf := make([]byte, 1024)
	c := 0
	for {
		n, _, err := connD.ReadFromUDP(buf)
		if err != nil {
			t.Fatal(err)
		}
		msg := string(buf[:n])
		if msg != expectedLog {
			t.Fatalf("Unexpected log:\nGot:\t\t%s\nExpected:\t%s\n", msg, expectedLog)
		}
		c++
		if c == len(logs) {
			return
		}
	}
}

func TestForwardedHttp(t *testing.T) {
	// Setup forwarder
	log.SetLevel(log.InfoLevel)
	listenAddr := "127.0.0.1:26999"
	forwardAddr := "127.0.0.1:27500"
	prependStr := "foo"
	forwarder, err := Forward(listenAddr, forwardAddr, DefaultTimeout, prependStr)
	if forwarder == nil || err != nil {
		t.Fatal(err)
	}
	defer forwarder.Close()

	// Gameserver sends 3 logs (each with 2 lines) to the forwarder, each to forwarder
	log := "10/11/2019 - 23:41:02: Started map \"awp_city\" (CRC \"-2134348459\")"
	logs := []string{
		log + "\n" + log + "\n", // Multline log
		log,                     // Single line log
		"\n",                    // Empty line
		" \r\n",                 // Whitespace line
	}
	go func(tb testing.TB) {
		time.Sleep(time.Millisecond * 300) // Wait for daemon to be up
		url := "http://" + listenAddr
		for _, l := range logs {
			http.Post(url, "application/json", bytes.NewBuffer([]byte(l)))
		}
	}(t)

	// Daemon receives log from forwarder
	expectedLog := fmt.Sprintf("%s%s%s", prependStr, "L ", log)
	addr, err := net.ResolveUDPAddr("udp", forwardAddr)
	connD, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer connD.Close()
	buf := make([]byte, 1024)
	c := 0
	for {
		n, _, err := connD.ReadFromUDP(buf)
		if err != nil {
			t.Fatal(err)
		}
		msg := string(buf[:n])
		if msg != expectedLog {
			t.Fatalf("Unexpected log:\nGot:\t\t%s\nExpected:\t%s\n", msg, expectedLog)
		}
		c++
		if c == 3 { // Should only receive 3 log lines
			return
		}
	}
}
