package main

import (
	"flag"
	log "github.com/sirupsen/logrus"
	"os"
	"strconv"
	"fmt"
	"strings"
	"regexp"
	"runtime"

	udpforwarder "source-udp-forwarder/pkg/forwarder"
)

var (
	// VERSION, BUILD_DATE, GIT_COMMIT are filled in by the build script
	VERSION     = "<Will be added by go build>"
	BUILD_DATE  = "<Will be added by go build>"
	COMMIT_SHA1 = "<Will be added by go build>"
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
	log.Info("Hello Walrus before FullTimestamp=true")

	var (
		listenAddress            = flag.String("udp.listen-address", getEnv("LISTEN_ADDR", ":26999"), "<IP>:<Port> to listen on for incoming packets.")
		forwardAddress			 = flag.String("udp.forward-address", getEnv("FORWARD_ADDR", "1.2.3.4:1013"), "<IP>:<Port> to which incoming packets will be forwarded.")

		proxyKey 				 = flag.String("forward.proxy-key", getEnv("PROXY_KEY", "XXXXX"), "The PROXY_KEY secret defined in HLStatsX:CE settings.")
		srcIp 					 = flag.String("forward.gameserver-ip", getEnv("GAMESERVER_IP", "127.0.0.1"), "IP that the sent packet should include")
		srcPort 				 = flag.String("forward.gameserver-port", getEnv("GAMESERVER_PORT", "27015"), "Port that the sent packet should include.")

		// metricPath               = flag.String("web.telemetry-path", getEnv("WEB_TELEMETRY_PATH", "/metrics"), "Path under which to expose metrics.")
		logLevel                  = flag.String("log.level", getEnv("LOG_LEVEL", "INFO"), "Output verbose debug information")
		logFormat                = flag.String("log.format", getEnv("LOG_FORMAT", "txt"), "Log format, valid options are txt and json")
		showVersion              = flag.Bool("version", false, "Show version information and exit")
	)
	flag.Parse()

	switch *logFormat {
		case "json":
			log.SetFormatter(&log.JSONFormatter{})
		default:
			log.SetFormatter(&log.TextFormatter{})
	}

	log.Printf("Source Builder Exporter %s, build date: %s, Commit SHA: %s, Go version: %s", VERSION, BUILD_DATE, COMMIT_SHA1, runtime.Version())

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
	log.SetLevel(log.DebugLevel)
	log.Printf("Log level: %s", *logLevel)
	log.Printf("Log format: %s", *logFormat)

	log.Printf("Listen Addr: %s", *forwardAddress)
	log.Printf("Forward Addr: %s", *listenAddress)
	re := regexp.MustCompile(".")
	log.Infof("Proxy Key: %s", re.ReplaceAllString(*proxyKey, "*"))
	log.Infof("Gameserver IP: %s", *srcIp)
	log.Infof("Gameserver Port: %s", *srcPort)

	if *showVersion {
		return
	}

	forwarder, err := udpforwarder.Forward(*listenAddress, *forwardAddress, udpforwarder.DefaultTimeout, fmt.Sprintf("PROXY Key=%s %s:%sPROXY ", *proxyKey, *srcIp, *srcPort))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Block forever
	select {}

	// Stop the forwarder
	forwarder.Close()
}

