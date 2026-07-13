package cluster

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/nightzjp/kafka-manager/internal/config"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"github.com/twmb/franz-go/pkg/sasl/scram"
)

type KafkaClient struct { Client *kgo.Client }

func (c *KafkaClient) Ping(ctx context.Context) error { return c.Client.Ping(ctx) }
func (c *KafkaClient) Close() { c.Client.Close() }

type KafkaFactory struct{}

func (KafkaFactory) Create(_ context.Context, cfg config.ClusterConfig) (Client, error) {
	opts, err := Options(cfg)
	if err != nil { return nil, err }
	client, err := kgo.NewClient(opts...)
	if err != nil { return nil, fmt.Errorf("create Kafka client: %w", err) }
	return &KafkaClient{Client: client}, nil
}

func Options(cfg config.ClusterConfig) ([]kgo.Opt, error) {
	opts := []kgo.Opt{kgo.SeedBrokers(cfg.Brokers...), kgo.ClientID("kafka-manager")}
	security := cfg.Security
	protocol := security.Protocol
	if protocol == "" { protocol = "PLAINTEXT" }
	if protocol == "SSL" || protocol == "SASL_SSL" || security.TLS {
		opts = append(opts, kgo.DialTLSConfig(&tls.Config{MinVersion: tls.VersionTLS12}))
	}
	if protocol == "SASL_PLAINTEXT" || protocol == "SASL_SSL" {
		switch security.Mechanism {
		case "PLAIN":
			opts = append(opts, kgo.SASL(plain.Auth{User: security.Username, Pass: security.Password}.AsMechanism()))
		case "SCRAM-SHA-256":
			opts = append(opts, kgo.SASL(scram.Auth{User: security.Username, Pass: security.Password}.AsSha256Mechanism()))
		case "SCRAM-SHA-512":
			opts = append(opts, kgo.SASL(scram.Auth{User: security.Username, Pass: security.Password}.AsSha512Mechanism()))
		default:
			return nil, fmt.Errorf("unsupported SASL mechanism %q", security.Mechanism)
		}
	}
	return opts, nil
}
