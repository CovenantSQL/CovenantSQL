package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/CovenantSQL/CovenantSQL/crypto/asymmetric"
	"github.com/CovenantSQL/CovenantSQL/crypto/kms"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/sys/unix"
)

var (
	version = "unknown"
	configFile string
	privateKeyFile string
	privateKey *asymmetric.PrivateKey
)

func init() {
	log.SetLevel(log.InfoLevel)
	flag.StringVar(&configFile, "config", "./config.yaml", "config file path (default: ./config.yaml)")
	flag.StringVar(&privateKeyFile, "private", "./private.key", "private key path (default: ./private.key)")
}

func main() {
	flag.Parse()

	// load private key
	masterKey, err := readMasterKey()
	if err != nil {
		log.Errorf("Unexpected error: %v\n", err)
		os.Exit(1)
	}
	privateKey, err = kms.LoadPrivateKey(privateKeyFile, []byte(masterKey))
	if err != nil {
		log.Fatalf("load private key file fail: %v\n", err)
	}

	// load the config
	c, err := LoadConfig(configFile)
	if err != nil {
		log.Errorf("Unexpected error: %v\n", err)
		os.Exit(1)
	}

	// connect ethereum client
	ec, err := NewEthereumClient(c)
	if err != nil {
		log.Errorf("Unexpected error: %v\n", err)
		os.Exit(1)
	}

	// listen the ethereum network
	go ec.ListenEvent(ethHandler)
	go ec.ListenHeader()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, unix.SIGTERM)
	<-stop
	close(ec.stopCh)
}

func readMasterKey() (string, error) {
	fmt.Println("Enter master key(press Enter for default: \"\"): ")
	bytePwd, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	return string(bytePwd), err
}
