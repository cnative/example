package cmd

import (
	"github.com/urfave/cli"
)

var (
	// TLSCertFile used to specify TLS Certificate file
	TLSCertFile = cli.StringFlag{
		Name:   "tls-cert-file",
		Usage:  "x509 server certificate for TLS",
		EnvVar: "TLS_CERT_FILE",
	}

	// TLSPrivateKeyFile used to specify TLS Private Key File
	TLSPrivateKeyFile = cli.StringFlag{
		Name:   "tls-private-key-file",
		Usage:  "x509 private key matching --tls-cert-file",
		EnvVar: "TLS_CERT_PRIVATE_KEY_FILE",
	}

	// TLSCertDir used to point to directory containing TLS Certificate and Private Key files
	TLSCertDir = cli.StringFlag{
		Name:   "tls-cert-dir",
		Usage:  "directory where the TLS certs are located. If --tls-cert-file and --tls-private-key-file are provided, this flag will be ignored",
		EnvVar: "TLS_CERT_DIR",
	}

	// ClientCAFile points to file that is used to validate/verify Clients in a TLS handshake
	ClientCAFile = cli.StringFlag{
		Name:   "client-ca-file",
		Usage:  "if set, any request presenting a client certificate signed by one of the authorities in the client-ca-file is authenticated with an identity corresponding to the CommonName of the client certificate",
		EnvVar: "CLIENT_CA_FILE",
	}

	// RootCAFile points to file that is used to validate/verify Servers in a TLS handshake
	RootCAFile = cli.StringFlag{
		Name:   "root-ca-file",
		Usage:  "path to a cert file for the certificate authority used to verify server",
		EnvVar: "ROOT_CA_FILE",
	}

	// InsecureSkipTLS flag to turn off TLS for server
	InsecureSkipTLS = cli.BoolFlag{
		Name:   "insecure-skip-tls",
		Hidden: true,
		Usage:  "used only for dev purpose. start the server without TLS",
	}

	// InsecureSkipVerifyTLS controls whether a client verifies the
	// server's certificate chain and host name.
	// If InsecureSkipVerify is true, TLS accepts any certificate
	// presented by the server and any host name in that certificate.
	// In this mode, TLS is susceptible to man-in-the-middle attacks.
	// This should be used only for testing.
	InsecureSkipVerifyTLS = cli.BoolFlag{
		Name:   "insecure-skip-verify-tls",
		Hidden: true,
		Usage:  "used only for dev purpose. client ignores server host name verification",
	}

	// Labels flag to specify labels
	Labels = cli.StringSliceFlag{
		Name:  "label",
		Usage: "name value pairs of the format key=value. repeat this flag to specify multiple label.",
	}

	// OutputFormat flag to specify output format. possible values yaml/json
	OutputFormat = cli.StringFlag{
		Name:  "output-format",
		Usage: "response output format",
		Value: "yaml",
	}

	// DBHost database host
	DBHost = cli.StringFlag{
		Name:   "db-host",
		Usage:  "database host",
		EnvVar: "DB_HOST",
		Value:  "localhost",
	}

	//DBPort database port
	DBPort = cli.UintFlag{
		Name:   "db-port",
		Usage:  "database port",
		EnvVar: "DB_PORT",
		Value:  5432,
	}

	//DBName database name
	DBName = cli.StringFlag{
		Name:   "db-name",
		Usage:  "database name",
		EnvVar: "DB_NAME",
		Value:  "reports",
	}

	//DBUser database user
	DBUser = cli.StringFlag{
		Name:   "db-user",
		Usage:  "database user",
		EnvVar: "DB_USER",
		Value:  "postgres",
	}

	//DBPassword database password
	DBPassword = cli.StringFlag{
		Name:   "db-password",
		Usage:  "database password",
		EnvVar: "DB_PASSWORD",
	}

	// Debug enable debug logging
	Debug = cli.BoolFlag{
		Name:   "debug",
		Usage:  "Enable debug logging",
		EnvVar: "DEBUG",
	}

	// SkipProcessMetrics skip collecting process metrics
	SkipProcessMetrics = cli.BoolFlag{
		Name:   "skip-process-metrics",
		Usage:  "skip collecting process metrics",
		EnvVar: "SKIP_PROCESS_METRICS",
	}

	//PProfEnable flag to enable server runtime pprof server profile data
	PProfEnable = cli.BoolFlag{
		Name:   "enable-pprof",
		Hidden: true,
		Usage:  "Enable server runtime profiling data",
	}
)
