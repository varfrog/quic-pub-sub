package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/panjf2000/ants/v2"
	"github.com/pkg/errors"
	"github.com/quic-go/quic-go"
	"github.com/varfrog/quicpubsub/pkg/quichelper"
	"github.com/varfrog/quicpubsub/server/internal/app"
	"github.com/varfrog/quicpubsub/server/internal/transport"
	"go.uber.org/zap"
	"log"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// runConfig represents configuration needed to run this app.
type runConfig struct {
	Help                 bool   // Prints usage and exists if true
	TLSCertPemPath       string // Path to TLS cert.pem
	TLSPrivateKeyPath    string // Path to TLS private.key
	PublisherListenPort  int    // Port to listen on for publisher connections
	SubscriberListenPort int    // Port to listen on for subscriber connections
	MaxConnections       int    // Max number of simulteneous connections (type int required by ants)
	MaxMessageBytes      int    // Max number of bytes per RPC message (type int required by io.Reader)
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

	tlsConfig, err := buildTLSConfig(config.TLSCertPemPath, config.TLSPrivateKeyPath)
	if err != nil {
		log.Fatalf("buildTLSConfig: %v", err)
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("zap.NewDevelopment: %v", err)
	}

	connectionPool, err := ants.NewPool(config.MaxConnections)
	if err != nil {
		log.Fatalf("ants.NewPool: %v", err)
	}
	defer ants.Release()

	subscriberPool := app.NewSubscriberPool()
	publisherPool := app.NewPublisherPool()
	observer := app.NewObserver(publisherPool, subscriberPool, logger.Named("Observer"))
	pinger := quichelper.NewPinger(quichelper.NewDefaultPingerConfig(), logger)

	wg := sync.WaitGroup{}

	// Serve Publishers
	wg.Add(1)
	go func() {
		defer wg.Done()
		server := transport.NewQUICPubServer(
			transport.QUICPubServerConfig{
				TLSConfig:           tlsConfig,
				PublisherListenPort: config.PublisherListenPort,
				MaxMessageBytes:     config.MaxMessageBytes,
				PingTimeout:         time.Second * 10,
			},
			connectionPool,
			observer,
			pinger,
			logger.Named("QUICPubServer"))

		err = server.AcceptPublishers(
			ctx,
			quic.Config{
				MaxIdleTimeout:        math.MaxInt64, // We ping, so no need for this
				MaxIncomingStreams:    int64(config.MaxConnections),
				MaxIncomingUniStreams: int64(config.MaxConnections),
			})
		if err != nil {
			fmt.Printf("AcceptPublishers: %v", err)
		}
	}()

	// Serve Subscribers
	wg.Add(1)
	go func() {
		defer wg.Done()
		server := transport.NewQUICSubServer(
			transport.QUICSubServerConfig{
				TLSConfig:            tlsConfig,
				SubscriberListenPort: config.SubscriberListenPort,
				MaxMessageBytes:      config.MaxMessageBytes,
			},
			connectionPool,
			observer,
			pinger,
			logger.Named("QUICSubServer"))

		err = server.AcceptSubscribers(ctx, quic.Config{
			MaxIdleTimeout:        math.MaxInt64, // We ping, so no need for this
			MaxIncomingStreams:    int64(config.MaxConnections),
			MaxIncomingUniStreams: int64(config.MaxConnections),
		})
		if err != nil {
			fmt.Printf("AcceptSubscribers: %v", err)
		}
	}()

	wg.Wait()
}

// parseFlagsIntoConfig gets a config needed to run this app from command-line flags.
func parseFlagsIntoConfig() (runConfig, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	var (
		help                 bool
		certPemPath          string
		privateKeyPath       string
		publisherListenPort  int
		subscriberListenPort int
		maxConnections       int
		maxMessageBytes      int
	)

	flag.BoolVar(&help, "help", false, "Print usage information")
	flag.StringVar(&certPemPath, "cert", filepath.Join(workingDir, "certs", "cert.pem"), "Path to TLS cert.pem")
	flag.StringVar(&privateKeyPath, "key", filepath.Join(workingDir, "certs", "private.key"), "Path to TLS private.key")
	flag.IntVar(&publisherListenPort, "pub-in-port", 5000, "Port to listen on for publisher connections")
	flag.IntVar(&subscriberListenPort, "sub-in-port", 5001, "Port to listen on for subscriber connections")
	flag.IntVar(&maxConnections, "max-connections", 10000, "Max number of simultaneous connections")
	flag.IntVar(&maxMessageBytes, "max-message-bytes", 1000, "Max number of bytes per message")
	flag.Parse()

	return runConfig{
		Help:                 help,
		TLSCertPemPath:       certPemPath,
		TLSPrivateKeyPath:    privateKeyPath,
		PublisherListenPort:  publisherListenPort,
		SubscriberListenPort: subscriberListenPort,
		MaxConnections:       maxConnections,
		MaxMessageBytes:      maxMessageBytes,
	}, nil
}

func validateRunConfig(config runConfig) error {
	if config.MaxConnections < 1 {
		return errors.New("MaxConnections < 1, serving requests is impossible")
	}
	if config.MaxMessageBytes < 1 {
		return errors.New("MaxMessageBytes < 1")
	}
	if _, err := os.Stat(config.TLSCertPemPath); errors.Is(err, os.ErrNotExist) {
		return errors.New("cannot stat the TLS cert.pem file, change the working dir to the project root or specify flag -cert")
	}
	if _, err := os.Stat(config.TLSPrivateKeyPath); errors.Is(err, os.ErrNotExist) {
		return errors.New("cannot stat the TLS private key file, change the working dir to the project root or specify flag -key")
	}
	return nil
}

func buildTLSConfig(certFilePath string, privateKeyFilePath string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFilePath, privateKeyFilePath)
	if err != nil {
		return nil, errors.Wrap(err, "tls.LoadX509KeyPair")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
	}, nil
}
