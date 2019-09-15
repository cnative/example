package cmd

import (
	"strings"

	"github.com/urfave/cli"

	"github.com/cnative/example/pkg/auth"
)

var (
	//OIDCFlags Open ID connect flags
	OIDCFlags = []cli.Flag{

		// OIDCIssuerURL flag to specify  oidc issuer url
		cli.StringFlag{
			Name:   "oidc-issuer-url",
			Usage:  "URL of the OpenID issuer, only HTTPS scheme will be accepted. If set, it will be used to verify the OIDC JSON Web Token (JWT)",
			EnvVar: "OIDC_ISSUER_URL",
		},

		// OIDCClientID flag to specify oidc client id
		cli.StringFlag{
			Name:   "oidc-client-id",
			Usage:  "client ID for the OpenID Connect client, must be set if oidc-issuer-url is set",
			EnvVar: "OIDC_CLIENT_ID",
		},

		// OIDCCAFlag flag to specify root ca of oidc issuer server
		cli.StringFlag{
			Name:   "oidc-ca-file",
			Usage:  "If set, the OpenID server's certificate will be verified by one of the authorities in the oidc-ca-file, otherwise the host's root CA set will be used",
			EnvVar: "OIDC_CA_CERT_FILE",
		},

		// OIDCRequiredClaim flag to specify minimum required oidc claims
		cli.StringSliceFlag{
			Name:   "oidc-required-claim",
			Usage:  "If set, the claim, which is name value pairs of the format key=value is verified to be present in the ID Token with a matching value. repeat this flag to specify multiple label",
			EnvVar: "OIDC_REQUIRED_CLAIMS",
		},

		// OIDCSigningAlgos flag to specify signing algorithms to use
		cli.StringFlag{
			Name:   "oidc-signing-algos",
			Usage:  "JOSE asymmetric signing algorithms. JWTs with a 'alg' header value not in this list will be rejected. Values are defined by RFC 7518 https://tools.ietf.org/html/rfc7518#section-3.1",
			Value:  "RS256",
			EnvVar: "OIDC_SIGNING_ALGOS",
		},
	}
)

// OIDCAuthOptionsFromCLI get open id connect auth options from cli
func OIDCAuthOptionsFromCLI(c *cli.Context) []auth.Option {
	issuerURL := c.String("oidc-issuer-url")
	if issuerURL == "" {
		return nil
	}
	clientID := c.String("oidc-client-id")
	caFile := c.String("oidc-ca-file")

	signingAlgos := strings.Split(c.String("oidc-signing-algos"), ",")

	requiredClaims := map[string]string{}
	for _, l := range c.StringSlice("oidc-required-claim") {
		nv := strings.Split(l, "=")
		if len(nv) == 2 {
			requiredClaims[nv[0]] = nv[1]
		}
	}

	return []auth.Option{
		auth.OIDCIssuer(issuerURL),
		auth.OIDCAudience(clientID),
		auth.OIDCCAFile(caFile),
		auth.OIDCSigningAlgos(signingAlgos),
		auth.OIDCRequiredClaims(requiredClaims),
	}
}
