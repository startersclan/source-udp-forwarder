// Shamelessly copied from: https://github.com/1lann/udp-forward/blob/788b94a/forward.go, modified accordingly

package udpforwarder

import (
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const bufferSize = 1024

type connection struct {
	udp        *net.UDPConn
	lastActive time.Time
}

type Forwarder struct {
	src          *net.UDPAddr
	dst          *net.UDPAddr
	client       *net.UDPAddr
	listenerConn *net.UDPConn

	connections      map[string]connection
	connectionsMutex *sync.RWMutex

	prependStr      string
	prependStrBytes []byte

	connectCallback    func(addr string)
	disconnectCallback func(addr string)

	timeout time.Duration

	closed bool
}

// DefaultTimeout is the default timeout period of inactivity for convenience
// sake. It is equivelant to 5 minutes.
var DefaultTimeout = time.Minute * 5

func Forward(src, dst string, timeout time.Duration, prependStr string) (*Forwarder, error) {
	forwarder := new(Forwarder)
	forwarder.connectCallback = func(addr string) {}
	forwarder.disconnectCallback = func(addr string) {}
	forwarder.connectionsMutex = new(sync.RWMutex)
	forwarder.connections = make(map[string]connection)
	forwarder.timeout = timeout
	forwarder.prependStr = prependStr
	forwarder.prependStrBytes = []byte(prependStr)

	var err error
	forwarder.src, err = net.ResolveUDPAddr("udp", src)
	if err != nil {
		return nil, err
	}

	forwarder.dst, err = net.ResolveUDPAddr("udp", dst)
	if err != nil {
		return nil, err
	}

	forwarder.client = &net.UDPAddr{
		IP:   forwarder.src.IP,
		Port: 0,
		Zone: forwarder.src.Zone,
	}

	forwarder.listenerConn, err = net.ListenUDP("udp", forwarder.src)
	if err != nil {
		return nil, err
	}
	log.Infof("Listening on %s", forwarder.src)

	go forwarder.janitor()
	go forwarder.run()

	return forwarder, nil
}

func (f *Forwarder) run() {
	for {
		buf := make([]byte, bufferSize)
		n, addr, err := f.listenerConn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		log.Debugf("Received buf length: %d, string: %s", n, string(buf[:n]))
		log.Traceln(buf)

		log.Debugf("prependStrBytes len: %d, string: %s", len(buf[:n]), f.prependStr)
		log.Traceln(f.prependStrBytes)

		log.Debugf("Prepending prependStrBytes to buf")
		newbuf := append([]byte(f.prependStrBytes), buf[:n]...)
		log.Debugf("newbuf len: %d, string: %s", len(newbuf), string(newbuf))
		log.Traceln(newbuf)
		go f.handle(newbuf, addr)
	}
}

func (f *Forwarder) janitor() {
	for !f.closed {
		time.Sleep(f.timeout)
		var keysToDelete []string

		f.connectionsMutex.RLock()
		for k, conn := range f.connections {
			if conn.lastActive.Before(time.Now().Add(-f.timeout)) {
				keysToDelete = append(keysToDelete, k)
			}
		}
		f.connectionsMutex.RUnlock()

		f.connectionsMutex.Lock()
		for _, k := range keysToDelete {
			f.connections[k].udp.Close()
			log.Infof("Cleaning up unused connection: %s", k)
			delete(f.connections, k)
		}
		f.connectionsMutex.Unlock()

		for _, k := range keysToDelete {
			f.disconnectCallback(k)
		}
	}
}

func (f *Forwarder) handle(data []byte, addr *net.UDPAddr) {
	f.connectionsMutex.RLock()
	conn, found := f.connections[addr.String()]
	f.connectionsMutex.RUnlock()

	if !found {
		log.Infof("Client connection does not exist. Added connection: %s", addr.String())
		conn, err := net.ListenUDP("udp", f.client)
		if err != nil {
			log.Println("udp-forwarder: failed to dial:", err)
			return
		}
		log.Debugf("Listening on %s", conn.LocalAddr().String())

		f.connectionsMutex.Lock()
		f.connections[addr.String()] = connection{
			udp:        conn,
			lastActive: time.Now(),
		}
		f.connectionsMutex.Unlock()

		f.connectCallback(addr.String())

		log.Debugf("Forwarding data from %s to %s", conn.LocalAddr().String(), f.dst.String())
		conn.WriteTo(data, f.dst)

		for {
			buf := make([]byte, bufferSize)
			n, _, err := conn.ReadFromUDP(buf)
			if err != nil {
				log.Debugf("Closing %s", conn.LocalAddr().String())
				f.connectionsMutex.Lock()
				conn.Close()
				delete(f.connections, addr.String())
				f.connectionsMutex.Unlock()
				return
			}

			go func(data []byte, conn *net.UDPConn, addr *net.UDPAddr) {
				f.listenerConn.WriteTo(data, addr)
			}(buf[:n], conn, addr)
		}
	} else {
		log.Debugf("Reusing existing client connection: %s", addr.String())
	}

	log.Debugf("Forwarding data to: %s", f.dst.String())
	conn.udp.WriteTo(data, f.dst)

	shouldChangeTime := false
	f.connectionsMutex.RLock()
	if _, found := f.connections[addr.String()]; found {
		if f.connections[addr.String()].lastActive.Before(
			time.Now().Add(f.timeout / 4)) {
			shouldChangeTime = true
		}
	}
	f.connectionsMutex.RUnlock()

	if shouldChangeTime {
		f.connectionsMutex.Lock()
		// Make sure it still exists
		if _, found := f.connections[addr.String()]; found {
			connWrapper := f.connections[addr.String()]
			connWrapper.lastActive = time.Now()
			f.connections[addr.String()] = connWrapper
		}
		f.connectionsMutex.Unlock()
	}
}

// Close stops the forwarder.
func (f *Forwarder) Close() {
	f.connectionsMutex.Lock()
	f.closed = true
	for _, conn := range f.connections {
		conn.udp.Close()
	}
	f.listenerConn.Close()
	f.connectionsMutex.Unlock()
}

// OnConnect can be called with a callback function to be called whenever a
// new client connects.
func (f *Forwarder) OnConnect(callback func(addr string)) {
	f.connectCallback = callback
}

// OnDisconnect can be called with a callback function to be called whenever a
// new client disconnects (after 5 minutes of inactivity).
func (f *Forwarder) OnDisconnect(callback func(addr string)) {
	f.disconnectCallback = callback
}

// Connected returns the list of connected clients in IP:port form.
func (f *Forwarder) Connected() []string {
	f.connectionsMutex.Lock()
	defer f.connectionsMutex.Unlock()
	results := make([]string, 0, len(f.connections))
	for key := range f.connections {
		results = append(results, key)
	}
	return results
}
