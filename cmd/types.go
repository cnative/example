package main

import (
	"github.com/cnative/pkg/auth"
)

type (
	ocExporterConfig struct {
		traceEnabled bool
		host         string
		port         uint
		namespace    string
	}

	tlsConfig struct {
		certFile string
		keyFile  string
		caFile   string
		skip     bool
	}

	// base server config used by all servers
	serverConfig struct {
		ocAgent            ocExporterConfig
		tls                tlsConfig
		auth               authConfig
		debug              bool
		dPort              uint
		gPort              uint
		hPort              uint
		mPort              uint
		gwPort             uint
		gwEnabled          bool
		skipProcessMetrics bool
		tags               map[string]string
	}

	dbConfig struct {
		name     string
		host     string
		port     uint
		user     string
		password string
	}

	authConfig struct {
		disabled       bool
		issuerURL      string
		clientID       string
		caFile         string
		signingAlgos   []string
		requiredClaims map[string]string
	}
)

func (a authConfig) asRuntimeAuthOptions() (opts []auth.Option) {

	if a.disabled {
		return
	}

	if a.issuerURL != "" {
		opts = append(opts, auth.OIDCIssuer(a.issuerURL))
	}

	if a.clientID != "" {
		opts = append(opts, auth.OIDCAudience(a.clientID))
	}

	if a.caFile != "" {
		opts = append(opts, auth.OIDCCAFile(a.caFile))
	}

	if len(a.signingAlgos) > 0 {
		opts = append(opts, auth.OIDCSigningAlgos(a.signingAlgos))
	}
	if len(a.requiredClaims) > 0 {
		opts = append(opts, auth.OIDCRequiredClaims(a.requiredClaims))
	}

	return
}
