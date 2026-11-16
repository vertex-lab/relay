package relay

import (
	"errors"
	"fmt"

	"github.com/nbd-wtf/go-nostr"
)

type Config struct {
	Address       string `env:"RELAY_ADDRESS"`
	Domain        string `env:"RELAY_DOMAIN"` // the domain used for nip-42
	QueueCapacity int    `env:"RELAY_QUEUE_CAPACITY"`
	Processors    int    `env:"RELAY_PROCESSORS"`
	SecretKey     string `env:"RELAY_SECRET_KEY"`
	PublicKey     string `env:"-"`
	Name          string `env:"RELAY_NAME"`
	Description   string `env:"RELAY_DESCRIPTION"`
	Icon          string `env:"RELAY_ICON"`
	Contact       string `env:"RELAY_CONTACT"`
	// OperatorPublicKey is advertised in NIP-11, independently of the signing key.
	// An empty value falls back to PublicKey.
	OperatorPublicKey string `env:"RELAY_OPERATOR_PUBKEY"`
}

// NewConfig returns a relay configuration structure with default paramenters.
func NewConfig() Config {
	return Config{
		Address:       "localhost:3334",
		QueueCapacity: 1000,
		Processors:    4,
		Name:          "Vertex Relay",
		Description:   "Web of Trust Relay powered by Vertex",
		Icon:          "https://image.nostr.build/7afc9d727d6486851cc2fe09865e7cc383449f8bad1700a9508db4d2815b6f1a.png",
	}
}

// Init initializes the config, deriving the public key from the secret key.
func (c *Config) Init() error {
	pk, err := nostr.GetPublicKey(c.SecretKey)
	if err != nil {
		return fmt.Errorf("Init: secret key is invalid: %w", err)
	}

	c.PublicKey = pk
	return nil
}

func (c Config) Validate() error {
	if c.OperatorPublicKey != "" && !nostr.IsValidPublicKey(c.OperatorPublicKey) {
		return errors.New("operator public key is invalid")
	}
	if c.QueueCapacity < 0 {
		return fmt.Errorf("queue capacity value must be positive: %d", c.QueueCapacity)
	}
	if c.Processors < 0 {
		return fmt.Errorf("processors value must be positive: %d", c.Processors)
	}

	if !nostr.IsValid32ByteHex(c.SecretKey) {
		return errors.New("secret key is invalid")
	}
	pk, err := nostr.GetPublicKey(c.SecretKey)
	if err != nil {
		return fmt.Errorf("secret key is invalid: %w", err)
	}
	if pk != c.PublicKey {
		return fmt.Errorf("public key and secret key don't match")
	}
	return nil
}

func (c Config) String() string {
	return fmt.Sprintf(
		"Relay:\n"+
			"\tAddress: %s\n"+
			"\tDomain: %s\n"+
			"\tQueue Capacity: %d\n"+
			"\tProcessors: %d\n"+
			"\tSecretKey: %s\n"+
			"\tPublicKey: %s\n",
		c.Address,
		c.Domain,
		c.QueueCapacity,
		c.Processors,
		c.SecretKey[0:4]+"...REDACTED..."+c.SecretKey[len(c.SecretKey)-4:],
		c.PublicKey,
	)
}
