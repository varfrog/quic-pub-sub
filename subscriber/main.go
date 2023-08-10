package main

import (
	"context"
	"crypto/tls"
	"flag"
	"github.com/pkg/errors"
	"github.com/quic-go/quic-go"
	"github.com/varfrog/quicpubsub/pkg/quichelper"
	"github.com/varfrog/quicpubsub/subscriber/internal/transport"
	"go.uber.org/zap"
	"log"
	"math"
	"os"
	"path/filepath"
)

// Represents configuration needed to run this app.
type runConfig struct {
	Help            bool   // Prints usage and exists if true
	TLSCertsDir     string // Path to a directory containing TLS certificates
	ServerPort      int
	MaxMessageBytes int // Max number of bytes per RPC message (type int required by io.Reader)
}

func main() {
	ctx := context.Background()

	config, err := parseFlagsIntoConfig()
	if err != nil {
		log.Fatal(err)
	}

	if config.Help {
		flag.PrintDefaults()
		os.Exit(0)
	}

	if err := validateRunConfig(config); err != nil {
		log.Fatal(err)
	}

	tlsConfig, err := buildTLSConfig(config.TLSCertsDir)
	if err != nil {
		log.Fatalf("buildTLSConfig: %v", err)
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("zap.NewDevelopment: %v", err)
	}

	subscriber := transport.NewQUICSubscriber(
		transport.QUICSubscriberConfig{
			TLSConfig:       tlsConfig,
			ServerPort:      config.ServerPort,
			MaxMessageBytes: config.MaxMessageBytes,
		},
		quic.Config{MaxIdleTimeout: math.MaxInt64},
		quichelper.NewPinger(quichelper.NewDefaultPingerConfig(), logger),
		logger)

	err = subscriber.Run(ctx)
	if err != nil {
		log.Fatalf("Run: %v", err)
	}
}

// parseFlagsIntoConfig gets a config needed to run this app.
func parseFlagsIntoConfig() (runConfig, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	var (
		help            bool
		certPath        string
		serverPort      int
		maxMessageBytes int
	)

	flag.BoolVar(&help, "help", false, "Print usage information")
	flag.StringVar(&certPath, "cert-path", filepath.Join(workingDir, "certs"), "Path to certs dir")
	flag.IntVar(&serverPort, "server-port", 5001, "Server port")
	flag.IntVar(&maxMessageBytes, "max-message-bytes", 1000, "Max number of bytes per message")
	flag.Parse()

	return runConfig{
		Help:            help,
		TLSCertsDir:     certPath,
		ServerPort:      serverPort,
		MaxMessageBytes: maxMessageBytes,
	}, nil
}

func validateRunConfig(config runConfig) error {
	if config.MaxMessageBytes < 1 {
		return errors.New("MaxMessageBytes < 1")
	}
	if _, err := os.Stat(config.TLSCertsDir); errors.Is(err, os.ErrNotExist) {
		return errors.New("cannot stat the TLS certs dir, change the working dir to the project root or specify flag -cert-path")
	}
	return nil
}

func buildTLSConfig(certDirPath string) (*tls.Config, error) {
	rootCAs, err := quichelper.GetRootCertPool(certDirPath)
	if err != nil {
		return nil, errors.Wrap(err, "GetRootCertPool")
	}

	return &tls.Config{
		RootCAs:            rootCAs,
		InsecureSkipVerify: true,
	}, nil
}
