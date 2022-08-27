package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// configをsetupしてポインタを返すくん
func SetupTLSConfig(cfg TLSConfig) (*tls.Config, error) {
	var err error
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS13}
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		// NOTE: 「クライアントの*tls.Config には、RootCAs と Certificates を設定することで、サーバ の証明書を検証し、サーバがクライアントの証明書を検証できるように設定されます。」
		// ってあるけどここの判定 cfg.Server じゃだめなんだろうか
		tlsConfig.Certificates = make([]tls.Certificate, 1)
		tlsConfig.Certificates[0], err = tls.LoadX509KeyPair(
			cfg.CertFile,
			cfg.KeyFile,
		)
		if err != nil {
			return nil, err
		}
	}

	if cfg.CAFile != "" {
		b, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, err
		}

		ca := x509.NewCertPool()
		ok := ca.AppendCertsFromPEM([]byte(b))
		if !ok {
			return nil, fmt.Errorf(
				"failed to parse root certificate: %q",
				cfg.CAFile,
			)
		}

		if cfg.Server {
			// サーバーの証明書

			tlsConfig.ClientCAs = ca
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		} else {
			// クライアントの証明書

			tlsConfig.RootCAs = ca
		}
		tlsConfig.ServerName = cfg.ServerAddress
	}

	return tlsConfig, nil
}

type TLSConfig struct {
	CertFile      string
	KeyFile       string
	CAFile        string
	ServerAddress string
	Server        bool
}
