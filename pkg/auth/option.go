package auth

import (
	"github.com/cnative/example/pkg/log"
)

type (
	// Option configures choices
	Option interface {
		apply(*runtime)
	}
	optionFunc func(*runtime)
)

// Logger for runtime
func Logger(l *log.Logger) Option {
	return optionFunc(func(r *runtime) {
		r.logger = l.NamedLogger("auth")
	})
}

// OIDCIssuer OIDC token issuer
func OIDCIssuer(iss string) Option {
	return optionFunc(func(r *runtime) {
		r.issuer = iss
	})
}

// OIDCAudience OIDC Audience which is the OIDC Client ID
func OIDCAudience(aud string) Option {
	return optionFunc(func(r *runtime) {
		r.aud = aud
	})
}

// OIDCCAFile CA file
func OIDCCAFile(caFile string) Option {
	return optionFunc(func(r *runtime) {
		r.caFile = caFile
	})
}

// OIDCSigningAlgos OIDC Signing Algos
func OIDCSigningAlgos(signingAlgos []string) Option {
	return optionFunc(func(r *runtime) {
		r.signingAlgos = signingAlgos
	})
}

// OIDCRequiredClaims OIDC Required Claims
func OIDCRequiredClaims(requiredClaims map[string]string) Option {
	return optionFunc(func(r *runtime) {
		r.requiredClaims = requiredClaims
	})
}

// Authorizer performs authz for every request
func Authorizer(authorizer AuthorizerFn) Option {
	return optionFunc(func(r *runtime) {
		r.authorizer = authorizer
	})
}
