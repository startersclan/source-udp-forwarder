package udpforwarder

import (
	"fmt"
	"net"
	"testing"
	"time"
)

func TestForward(t *testing.T) {
	listenAddr := "127.0.0.1:26999"
	forwardAddr := "127.0.0.1:27500"
	forwarder, err := Forward(
		listenAddr,
		forwardAddr,
		DefaultTimeout,
		"foo",
	)
	if forwarder == nil || err != nil {
		t.Fatal(err)
	}
	defer forwarder.Close()
}

func TestHandleConnection(t *testing.T) {
	listenAddr := "127.0.0.1:26999"
	forwardAddr := "127.0.0.1:27500"
	forwarder, err := Forward(
		listenAddr,
		forwardAddr,
		DefaultTimeout,
		"foo",
	)
	if forwarder == nil || err != nil {
		t.Fatal(err)
	}
	defer forwarder.Close()

	message := "L 10/11/2019 - 23:41:02: Started map \"awp_city\" (CRC \"-2134348459\")"

	// Gameserver sends log to forwarder
	go func(tb testing.TB) {
		conn, err := net.Dial("udp", listenAddr)
		if err != nil {
			tb.Fatal(err)
		}
		defer conn.Close()

		if _, err := fmt.Fprintf(conn, message); err != nil {
			tb.Fatal(err)
		}
	}(t)

	// Allow the time for the connection to be added
	time.Sleep(time.Millisecond * 100)

	connected := forwarder.Connected()
	if len(connected) == 0 {
		t.Fatalf("New connection not added!")
	}
}

func TestJanitor(t *testing.T) {
	// Setup forwarder
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
	message := "L 10/11/2019 - 23:41:02: Started map \"awp_city\" (CRC \"-2134348459\")"
	go func(tb testing.TB) {
		conn, err := net.Dial("udp", listenAddr)
		if err != nil {
			tb.Fatal(err)
		}
		defer conn.Close()

		if _, err := fmt.Fprintf(conn, message); err != nil {
			tb.Fatal(err)
		}
	}(t)

	// Allow the janitor some time to cleanup
	time.Sleep(timeout * 100)

	connected := forwarder.Connected()
	if len(connected) > 0 {
		t.Fatalf("Stale connection not cleaned up: %s", connected[0])
	}
}

func TestForwardedMessage(t *testing.T) {
	// Setup forwarder
	listenAddr := "127.0.0.1:26999"
	forwardAddr := "127.0.0.1:27500"
	prependStr := "foo"
	forwarder, err := Forward(listenAddr, forwardAddr, DefaultTimeout, prependStr)
	if forwarder == nil || err != nil {
		t.Fatal(err)
	}
	defer forwarder.Close()

	// Gameserver sends log to forwarder
	message := "L 10/11/2019 - 23:41:02: Started map \"awp_city\" (CRC \"-2134348459\")"
	go func(tb testing.TB) {
		conn, err := net.Dial("udp", listenAddr)
		if err != nil {
			tb.Fatal(err)
		}
		defer conn.Close()

		if _, err := fmt.Fprintf(conn, message); err != nil {
			tb.Fatal(err)
		}
	}(t)

	// Daemon receives log from forwarder and test
	expectedMessage := prependStr + message
	addr, err := net.ResolveUDPAddr("udp", forwardAddr)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	bufferSize := 1024
	buf := make([]byte, bufferSize)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			t.Fatal(err)
		}
		msg := string(buf[:n])
		if msg != expectedMessage {
			t.Fatalf("Unexpected message:\nGot:\t\t%s\nExpected:\t%s\n", msg, expectedMessage)
		}
		return
	}
}
