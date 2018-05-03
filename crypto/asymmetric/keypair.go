package asymmetric

import (
	"github.com/btcsuite/btcd/btcec"
	log "github.com/sirupsen/logrus"
)

func GenSecp256k1Keypair() (privateKey *btcec.PrivateKey, publicKey *btcec.PublicKey, err error) {
	privateKey, err = btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		log.Errorf("private key generation error: %s", err)
		return nil, nil, err
	}
	publicKey = privateKey.PubKey()
	return
}
