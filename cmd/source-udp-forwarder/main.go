package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	udpforwarder "github.com/startersclan/source-udp-forwarder/pkg/forwarder"
	version "github.com/startersclan/source-udp-forwarder/pkg/version"
)

func getEnv(key string, defaultVal string) string {
	if envVal, ok := os.LookupEnv(key); ok {
		return envVal
	}
	return defaultVal
}

func getEnvBool(key string) (envValBool bool) {
	if envVal, ok := os.LookupEnv(key); ok {
		envValBool, _ = strconv.ParseBool(envVal)
	}
	return
}

func main() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)
	// log.Info("Hello Walrus before FullTimestamp=true")

	var (
		listenAddress    = flag.String("listen-address", getEnv("LISTEN_ADDR", ":26999"), "<IP>:<PORT> to listen for incoming HTTP and UDP logs.")
		udpListenAddress = flag.String("udp.listen-address", getEnv("UDP_LISTEN_ADDR", ":26999"), "<IP>:<PORT> to listen for incoming HTTP and UDP logs. (deprecated, use -listen-address instead)")
		forwardAddress   = flag.String("udp.forward-address", getEnv("UDP_FORWARD_ADDR", "127.0.0.1:27500"), "<IP>:<PORT> of the daemon to which incoming packets will be forwarded.")

		proxyKey = flag.String("forward.proxy-key", getEnv("FORWARD_PROXY_KEY", "XXXXX"), "The PROXY_KEY secret defined in HLStatsX:CE settings.")
		srcIp    = flag.String("forward.gameserver-ip", getEnv("FORWARD_GAMESERVER_IP", "127.0.0.1"), "IP that the sent packet should include.")
		srcPort  = flag.String("forward.gameserver-port", getEnv("FORWARD_GAMESERVER_PORT", "27015"), "Port that the sent packet should include.")

		// metricPath               = flag.String("web.telemetry-path", getEnv("WEB_TELEMETRY_PATH", "/metrics"), "Path under which to expose metrics.")
		logLevel    = flag.String("log.level", getEnv("LOG_LEVEL", "INFO"), "Log level. Defaults to 'INFO'")
		logFormat   = flag.String("log.format", getEnv("LOG_FORMAT", "txt"), "Log format, valid options are txt and json")
		showVersion = flag.Bool("version", false, "Show version information and exit")
	)
	flag.Parse()

	// Support the deprecated flag
	if *udpListenAddress != ":26999" {
		listenAddress = udpListenAddress
	}

	switch *logFormat {
	case "json":
		log.SetFormatter(&log.JSONFormatter{})
	default:
		log.SetFormatter(&log.TextFormatter{})
	}

	log.Printf(version.GetVersion())
	if *showVersion {
		os.Exit(0)
	}

	*logLevel = strings.ToUpper(*logLevel)
	switch *logLevel {
	case "TRACE":
		log.SetLevel(log.TraceLevel)
	case "DEBUG":
		log.SetLevel(log.DebugLevel)
	case "INFO":
		log.SetLevel(log.InfoLevel)
	case "WARN":
		log.SetLevel(log.WarnLevel)
	case "ERROR":
		log.SetLevel(log.ErrorLevel)
	case "FATAL":
		log.SetLevel(log.FatalLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
	log.Printf("Log level: %s", *logLevel)
	log.Printf("Log format: %s", *logFormat)

	log.Printf("Listen Addr: %s", *listenAddress)
	log.Printf("Forward Addr: %s", *forwardAddress)
	re := regexp.MustCompile(".")
	log.Infof("Forward Proxy Key: %s", re.ReplaceAllString(*proxyKey, "*"))
	log.Infof("Forward Gameserver IP: %s", *srcIp)
	log.Infof("Forward Gameserver Port: %s", *srcPort)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGHUP)
	forwarder, err := udpforwarder.Forward(*listenAddress, *forwardAddress, udpforwarder.DefaultTimeout, fmt.Sprintf("PROXY Key=%s %s:%sPROXY ", *proxyKey, *srcIp, *srcPort))
	if forwarder == nil || err != nil {
		log.Fatal(err)
	}

	<-sig
	log.Infof("Received shutdown signal. Exiting")
	forwarder.Close()
	time.Sleep(10000)
}
