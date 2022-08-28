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
		//
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

		// ClientCAs defines the set of root certificate authorities
		// that servers use if required to verify a client certificate
		// by the policy in ClientAuth.
		ca := x509.NewCertPool()
		// pemファイルから証明書を追加
		ok := ca.AppendCertsFromPEM([]byte(b))
		if !ok {
			return nil, fmt.Errorf(
				"failed to parse root certificate: %q",
				cfg.CAFile,
			)
		}

		if cfg.Server {
			// サーバー用の証明書であればClientCAsを設定してクライアントの証明書を検証できるようにする
			// ClientCAs defines the set of root certificate authorities
			// that servers use if required to verify a client certificate
			// by the policy in ClientAuth.
			tlsConfig.ClientCAs = ca
			// ClientAuth determines the server's policy for
			// TLS Client Authentication. The default is NoClientCert.
			// クライアントの証明書を検証する設定にする
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		} else {
			// クライアント用の証明書であればRootCAsを設定することでサーバの証明書を検証できるようにする
			// RootCAs defines the set of root certificate authorities
			// that clients use when verifying server certificates.
			// If RootCAs is nil, TLS uses the host's root CA set.
			tlsConfig.RootCAs = ca
		}
		// サーバー名は、返された証明書のホスト名を検証するために使用される
		// ServerName is used to verify the hostname on the returned
		// certificates unless InsecureSkipVerify is given. It is also included
		// in the client's handshake to support virtual hosting unless it is
		// an IP address.
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
