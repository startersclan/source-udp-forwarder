// Shamelessly copied from: https://github.com/1lann/udp-forward/blob/788b94a/forward.go, modified accordingly

package udpforwarder

import (
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
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
	src             *net.UDPAddr // UDP server
	dst             *net.UDPAddr // UDP destination
	client          *net.UDPAddr // My UDP client to forward to UDP destination
	listenerConnUdp *net.UDPConn

	connections      map[string]connection
	connectionsMutex *sync.RWMutex

	prependStr      string
	prependStrBytes []byte

	connectCallback    func(addr string)
	disconnectCallback func(addr string)

	timeout time.Duration

	closed bool

	httpSrv http.Server // HTTP server
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

	log.Infof("Listening on %s for UDP and HTTP logs", forwarder.src.String())

	// Listen for UDP
	forwarder.listenerConnUdp, err = net.ListenUDP("udp", forwarder.src)
	if err != nil {
		return nil, err
	}
	go forwarder.janitor()
	go forwarder.run()

	// Listen for TCP (HTTP)
	go func() {
		// sm := http.NewServeMux()
		// sm.HandleFunc("/", forwarder.httpHandler)
		forwarder.httpSrv = http.Server{
			Addr:    forwarder.src.String(),
			Handler: http.HandlerFunc(forwarder.httpHandler),
		}
		if err := forwarder.httpSrv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("HTTP server ListenAndServe: %v", err)
		}
	}()

	return forwarder, nil
}
func (f *Forwarder) httpHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case "GET":
		log.Debugf("Ignoring GET request")
	case "POST":
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal(err)
		}
		lines := strings.Split(string(reqBody[:]), "\n")
		for _, l := range lines {
			buf := []byte(l)
			n := len(buf)

			log.Debugf("[HTTP] Received log from %s. buf length: %d, string: %s", r.RemoteAddr, n, string(buf[:n]))
			log.Traceln(buf)

			log.Debugf("[HTTP] prependStrBytes len: %d, string: %s", len(f.prependStr), f.prependStr)
			log.Traceln(f.prependStrBytes)

			log.Debugf("[HTTP] Prepending prependStrBytes to buf")
			newbuf := append([]byte(f.prependStrBytes), buf[:n]...)
			log.Debugf("[HTTP] newbuf len: %d, string: %s", len(newbuf), string(newbuf))
			log.Traceln(newbuf)
			go f.handle(newbuf, nil, r.RemoteAddr)
		}
	default:
		log.Debugf("Invalid request")
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(http.StatusText(http.StatusNotImplemented)))
	}
}

func (f *Forwarder) run() {
	for {
		buf := make([]byte, bufferSize)
		n, addr, err := f.listenerConnUdp.ReadFromUDP(buf)
		if err != nil {
			return
		}
		log.Debugf("[UDP] Received log from %s. buf length: %d, string: %s", addr, n, string(buf[:n]))
		log.Traceln(buf)

		log.Debugf("[UDP] prependStrBytes len: %d, string: %s", len(f.prependStr), f.prependStr)
		log.Traceln(f.prependStrBytes)

		log.Debugf("[UDP] Prepending prependStrBytes to buf")
		newbuf := append([]byte(f.prependStrBytes), buf[:n]...)
		log.Debugf("[UDP] newbuf len: %d, string: %s", len(newbuf), string(newbuf))
		log.Traceln(newbuf)
		go f.handle(newbuf, addr, "")
	}
}

func (f *Forwarder) janitor() {
	for {
		f.connectionsMutex.Lock()
		if f.closed {
			return
		}
		f.connectionsMutex.Unlock()

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

func (f *Forwarder) handle(data []byte, clientUdpAddr *net.UDPAddr, clientTcpAddr string) {
	cAddr := ""
	if clientUdpAddr != nil {
		cAddr = clientUdpAddr.String()
	} else if clientTcpAddr != "" {
		cAddr = clientTcpAddr
	} else {
		log.Println("udp-forwarder: No clientUdpAddr and no clientTcpAddr")
		return
	}

	f.connectionsMutex.RLock()
	conn, found := f.connections[cAddr]
	f.connectionsMutex.RUnlock()

	if !found {
		log.Infof("Client connection does not exist. Added connection: %s", cAddr)
		conn, err := net.ListenUDP("udp", f.client)
		if err != nil {
			log.Println("udp-forwarder: failed to dial:", err)
			return
		}

		f.connectionsMutex.Lock()
		f.connections[cAddr] = connection{
			udp:        conn,
			lastActive: time.Now(),
		}
		f.connectionsMutex.Unlock()

		f.connectCallback(cAddr)

		log.Debugf("Forwarding log from %s to %s", conn.LocalAddr().String(), f.dst.String())
		conn.WriteTo(data, f.dst)

		if clientUdpAddr != nil {
			for {
				buf := make([]byte, bufferSize)
				_, _, err := conn.ReadFromUDP(buf)
				if err != nil {
					log.Debugf("Closing %s", conn.LocalAddr().String())
					f.connectionsMutex.Lock()
					conn.Close()
					delete(f.connections, cAddr)
					f.connectionsMutex.Unlock()
					return
				}

				// Reply to client?
				// go func(data []byte, conn *net.UDPConn, cAddr *net.UDPAddr) {
				// 	f.listenerConnUdp.WriteTo(data, cAddr)
				// }(buf[:n], conn, clientUdpAddr)
			}
		}
		if clientTcpAddr != "" {
			return
		}
	} else {
		log.Debugf("Reusing existing client connection: %s", conn.udp.LocalAddr())
	}

	log.Debugf("Forwarding log from %s to %s", conn.udp.LocalAddr(), f.dst.String())
	conn.udp.WriteTo(data, f.dst)

	shouldChangeTime := false
	f.connectionsMutex.RLock()
	if _, found := f.connections[cAddr]; found {
		if f.connections[cAddr].lastActive.Before(
			time.Now().Add(f.timeout / 4)) {
			shouldChangeTime = true
		}
	}
	f.connectionsMutex.RUnlock()

	if shouldChangeTime {
		f.connectionsMutex.Lock()
		// Make sure it still exists
		if _, found := f.connections[cAddr]; found {
			connWrapper := f.connections[cAddr]
			connWrapper.lastActive = time.Now()
			f.connections[cAddr] = connWrapper
		}
		f.connectionsMutex.Unlock()
	}
}

// Close stops the forwarder.
func (f *Forwarder) Close() {
	log.Infof("Stopping UDP server")
	f.connectionsMutex.Lock()
	f.closed = true
	for _, conn := range f.connections {
		conn.udp.Close()
	}
	f.listenerConnUdp.Close()
	f.connectionsMutex.Unlock()

	// See: https://pkg.go.dev/net/http#Server.Shutdown
	log.Infof("Stopping HTTP server")
	if err := f.httpSrv.Shutdown(context.Background()); err != nil {
		log.Printf("HTTP server Shutdown: %v", err)
	}
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
