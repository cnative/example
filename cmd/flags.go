package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func applyBaseServerConfig(cmd *cobra.Command) {

	cmd.PersistentFlags().BoolP("debug", "d", false, "start in debug mode")
	cmd.PersistentFlags().Int16("debug-port", debugPort, "debug port on which net/http/pprof data is exposed")

	cmd.PersistentFlags().Int16("health-port", healthPort, "health check port")
	cmd.PersistentFlags().Int16("metrics-port", metricsPort, "metrics port")
	cmd.PersistentFlags().Bool("skip-process-metrics", false, "skip collecting process metrics")
	cmd.PersistentFlags().StringSlice("tag", []string{}, "info attributes for server. name value pairs of the format key=value. repeat this flag to specify multiple label")
}

func applyTlSFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().Bool("insecure-skip-tls", false, "no tls. not recommended")
	cmd.PersistentFlags().String("tls-cert-file", "", "x509 server certificate for TLS")
	cmd.PersistentFlags().String("tls-private-key-file", "", "x509 private key matching --tls-cert-file")
	cmd.PersistentFlags().String("client-ca-file", "", "if set, any request presenting a client certificate signed by one of the authorities in the client-ca-file is authenticated with an identity corresponding to the CommonName of the client certificate")
}

func applyDBFlags(cmd *cobra.Command, store string) {
	prefix := fmt.Sprintf("%s-", store)
	cmd.PersistentFlags().String(fmt.Sprintf("%sdb-host", prefix), "localhost", fmt.Sprintf("%s store database host", store))
	cmd.PersistentFlags().Int(fmt.Sprintf("%sdb-port", prefix), 5432, fmt.Sprintf("%s store database port", store))
	cmd.PersistentFlags().String(fmt.Sprintf("%sdb-name", prefix), fmt.Sprintf("example-app-%s", store), fmt.Sprintf("%s store database name", store))
	cmd.PersistentFlags().String(fmt.Sprintf("%sdb-user", prefix), "postgres", fmt.Sprintf("%s store database user", store))
	cmd.PersistentFlags().String(fmt.Sprintf("%sdb-password", prefix), "", fmt.Sprintf("%s store database password ", store))
}

func applyOIDCFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().Bool("insecure-no-auth", false, "disable auth. not safe. do not use it.")
	cmd.PersistentFlags().String("oidc-issuer-url", "", "URL of the OpenID issuer, only HTTPS scheme will be accepted. If set, it will be used to verify the OIDC JSON Web Token (JWT)")
	cmd.PersistentFlags().String("oidc-client-id", "", "client ID for the OpenID Connect client, must be set if oidc-issuer-url is set")
	cmd.PersistentFlags().String("oidc-ca-file", "", "If set, the OpenID server's certificate will be verified by one of the authorities in the oidc-ca-file, otherwise the host's root CA set will be used")
	cmd.PersistentFlags().String("oidc-required-claim", "", "If set, the claim, which is name value pairs of the format key=value is verified to be present in the ID Token with a matching value. repeat this flag to specify multiple label")
	cmd.PersistentFlags().String("oidc-signing-algos", "", `JOSE asymmetric signing algorithms. JWTs with a 'alg' header value not in this list will be rejected. Values are defined by RFC 7518 https://tools.ietf.org/html/rfc7518#section-3.1 (default: "RS256")`)
}

func applyTraceConfig(cmd *cobra.Command) {
	cmd.PersistentFlags().Bool("no-trace", false, "disable tracing")
	cmd.PersistentFlags().String("oc-agent-host", "localhost", "opencensus agent host")
	cmd.PersistentFlags().Int("oc-agent-port", ocAgentPort, "opencensus agent host")
	cmd.PersistentFlags().String("oc-namespace", "", "opencensus service namespace")
}

func applyDBConfigs(dc *dbConfig, store string) {
	prefix := fmt.Sprintf("%s-", store)
	dc.name = viper.GetString(fmt.Sprintf("%sdb-name", prefix))
	dc.host = viper.GetString(fmt.Sprintf("%sdb-host", prefix))
	dc.port = uint(viper.GetInt(fmt.Sprintf("%sdb-port", prefix)))
	dc.user = viper.GetString(fmt.Sprintf("%sdb-user", prefix))
	dc.password = viper.GetString(fmt.Sprintf("%sdb-password", prefix))
}
