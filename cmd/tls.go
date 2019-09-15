package cmd

import (
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

// TLSConfig tls config Wrapper
type TLSConfig struct {
	CertFile string
	KeyFile  string
	CAFile   string
	skip     bool
}

// TLSConfigFromCLI get TLSConfig from the CLI wrapper
func TLSConfigFromCLI(c *cli.Context) (tls TLSConfig, err error) {

	keyFile := c.String(TLSPrivateKeyFile.Name)
	certFile := c.String(TLSCertFile.Name)
	tlsCertDir := c.String(TLSCertDir.Name)

	if keyFile == "" && certFile == "" && tlsCertDir != "" {
		keyFile = path.Join(tlsCertDir, "tls.key")
		certFile = path.Join(tlsCertDir, "tls.crt")
	}

	if (keyFile == "" || certFile == "") && !c.Bool(InsecureSkipTLS.Name) {
		err = errors.Errorf("TLS key and/or cert files not specified. use '--%s' or '--%s' / '--%s'", TLSCertDir.Name, TLSPrivateKeyFile.Name, TLSCertFile.Name)
		return
	}

	tls.CertFile = certFile
	tls.KeyFile = keyFile
	tls.CAFile = c.String(ClientCAFile.Name)
	tls.skip = c.Bool(InsecureSkipTLS.Name)

	return resolveAbsFilePath(tls)
}

func resolveAbsFilePath(tls TLSConfig) (TLSConfig, error) {

	if tls.CertFile != "" {
		f, err := filepath.Abs(tls.CertFile)
		if err != nil {
			return TLSConfig{}, err
		}
		tls.CertFile = f
	}

	if tls.KeyFile != "" {
		f, err := filepath.Abs(tls.KeyFile)
		if err != nil {
			return TLSConfig{}, err
		}
		tls.KeyFile = f
	}

	if tls.CAFile != "" {
		f, err := filepath.Abs(tls.CAFile)
		if err != nil {
			return TLSConfig{}, err
		}
		tls.CAFile = f
	}

	return tls, nil
}
