package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	udpforwarder "github.com/startersclan/source-udp-forwarder/pkg/forwarder"
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

func run() error {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)
	// log.Info("Hello Walrus before FullTimestamp=true")

	var (
		listenAddress  = flag.String("udp.listen-address", getEnv("UDP_LISTEN_ADDR", ":26999"), "<IP>:<Port> to listen on for incoming packets.")
		forwardAddress = flag.String("udp.forward-address", getEnv("UDP_FORWARD_ADDR", "1.2.3.4:1013"), "<IP>:<Port> to which incoming packets will be forwarded.")

		proxyKey = flag.String("forward.proxy-key", getEnv("FORWARD_PROXY_KEY", "XXXXX"), "The PROXY_KEY secret defined in HLStatsX:CE settings.")
		srcIp    = flag.String("forward.gameserver-ip", getEnv("FORWARD_GAMESERVER_IP", "127.0.0.1"), "IP that the sent packet should include.")
		srcPort  = flag.String("forward.gameserver-port", getEnv("FORWARD_GAMESERVER_PORT", "27015"), "Port that the sent packet should include.")

		// metricPath               = flag.String("web.telemetry-path", getEnv("WEB_TELEMETRY_PATH", "/metrics"), "Path under which to expose metrics.")
		logLevel    = flag.String("log.level", getEnv("LOG_LEVEL", "INFO"), "Log level. Defaults to 'INFO'")
		logFormat   = flag.String("log.format", getEnv("LOG_FORMAT", "txt"), "Log format, valid options are txt and json")
		showVersion = flag.Bool("version", false, "Show version information and exit")
	)
	flag.Parse()

	switch *logFormat {
	case "json":
		log.SetFormatter(&log.JSONFormatter{})
	default:
		log.SetFormatter(&log.TextFormatter{})
	}

	log.Printf(getVersion())
	if *showVersion {
		return nil
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

	forwarder, err := udpforwarder.Forward(*listenAddress, *forwardAddress, udpforwarder.DefaultTimeout, fmt.Sprintf("PROXY Key=%s %s:%sPROXY ", *proxyKey, *srcIp, *srcPort))
	if forwarder == nil || err != nil {
		return err
	}

	// Block forever
	select {}
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
