package auth

import (
	"context"
	"strings"

	"github.com/coreos/go-oidc"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/cnative/example/pkg/log"
)

// Resource is an object upon which an authorization check needs to be performed
// for ex.
//  		{"app": "cnative-example", "service": "baz", "name": "foo"}
//  		{"app": "cnative-example", "service": "baz", "name": "bar"}
//
type Resource struct {
	App     string `json:"app,omitempty"`
	Service string `json:"service,omitempty"`
	Name    string `json:"name,omitempty"`
}

// Action is an operation that can be performed on a Resource
// for ex.
// 		create, add-topics, set-bootstrap-brokers and filter are
//      the operation that are allowed on clusters resource
type Action string

// Claims represents the identity asserted by Authorization System
type Claims interface {
	GetSubject() string
	GetRoles() []string
}

// Runtime runtime authN/authZ
type Runtime interface {

	// Verifier authenticates & validates the token and establishes the Subject
	// Token is epected to be present in the context
	Verify(ctx context.Context) (context.Context, Claims, error)
	// Authorizer authorizes resource use
	Authorize(context.Context, Claims, Resource, Action) (context.Context, bool, error)
}

// AuthorizerFn is a function that authorizes each grpc requests.
type AuthorizerFn func(context.Context, Claims, Resource, Action) (bool, error)

type runtime struct {
	logger *log.Logger

	issuer         string            // oidc token issuer
	aud            string            // oidc audience
	caFile         string            // ca file
	requiredClaims map[string]string // oidc client ID
	signingAlgos   []string          // JOSE asymmetric signing algorithms
	authorizer     AuthorizerFn      // Authorizes each rpc call

	// TokenVerifier used to perform the JWT verification.
	verifier *oidc.IDTokenVerifier
}

type claims struct {
	token *oidc.IDToken
}

func (f optionFunc) apply(r *runtime) {
	f(r)
}

// NewRuntime returns a new Runtime
func NewRuntime(otions ...Option) (Runtime, error) {
	// setup defaults
	r := &runtime{}
	for _, opt := range otions {
		opt.apply(r)
	}
	if r.logger == nil {
		r.logger, _ = log.NewNop()
	}

	if r.issuer == "" {
		return nil, errors.New("token issuer not specified")
	}

	verifier, err := NewVerifier(context.Background(), r.issuer, r.aud)
	if err != nil {
		return nil, err
	}
	r.verifier = verifier

	r.logger.Infow("auth runtime initialized", "token-issuer", r.issuer, "audience", r.aud)

	return r, nil
}

func (r *runtime) Authorize(ctx context.Context, claims Claims, resource Resource, action Action) (context.Context, bool, error) {
	if r.authorizer != nil {
		result, err := r.authorizer(ctx, claims, resource, action)
		return ctx, result, err
	}

	return ctx, false, nil
}

func (r *runtime) Verify(ctx context.Context) (context.Context, Claims, error) {

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx, nil, status.Errorf(codes.Unauthenticated, "Context does not contain any metadata")
	}

	authHdrs := md.Get("authorization")
	if len(authHdrs) != 1 {
		return ctx, nil, errors.Errorf("Found %d authorization headers, expected 1", len(authHdrs))
	}

	sp := strings.SplitN(authHdrs[0], " ", 2)
	if len(sp) != 2 {
		return ctx, nil, errors.New("authorization header has is not '<type> <token> format")
	}
	if !strings.EqualFold(sp[0], "bearer") {
		return ctx, nil, errors.Errorf("Only bearer tokens are supported, not %s", sp[0])
	}

	tok, err := r.verifier.Verify(ctx, sp[1])
	if err != nil {
		return ctx, nil, err
	}

	return newContext(ctx, tok.Subject), &claims{tok}, nil
}

// GetSubject returns the sub field of this token
func (c *claims) GetSubject() string {

	return c.token.Subject
}

// GetSubject returns the sub field of this token
func (c *claims) GetRoles() []string {

	return []string{}
}

// NewVerifier returns a configured Verifier that can retrieve keys and verify
// tokens automatically from issuer. Audience is the client ID that we require
// the tokens to be issued from. If empty, this verification step will not be
// performed. The returned verifier is a long-lived process that can fetch and
// refetch keys as needed. The passed context can be used to cancel this, but it
// is expected to be alive for the duration of the verifier.
func NewVerifier(ctx context.Context, issuer, audience string) (*oidc.IDTokenVerifier, error) {
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, errors.Wrap(err, "Error creating OIDC provider")
	}

	var cfg oidc.Config
	if audience != "" {
		cfg.ClientID = audience
	} else {
		cfg.SkipClientIDCheck = true
	}

	return provider.Verifier(&cfg), nil
}
