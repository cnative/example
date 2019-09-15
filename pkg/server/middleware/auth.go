package middleware

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cnative/example/pkg/auth"
)

// Move this into Auth --> TODO
func auth0(ctx context.Context, authRuntime auth.Runtime) (context.Context, error) {

	ctx, c, err := authRuntime.Verify(ctx)
	if err != nil {
		return ctx, status.Errorf(codes.PermissionDenied, "%v", err.Error())
	}
	var r auth.Resource
	var a auth.Action
	ctx, allow, err := authRuntime.Authorize(ctx, c, r, a)
	if err != nil {
		return ctx, status.Errorf(codes.PermissionDenied, "resource=%v, action=%v, subject=%v, message=%v", r, a, c, err.Error())
	}

	if allow {
		return ctx, nil
	}

	return ctx, status.Errorf(codes.PermissionDenied, "resource=%v, action=%v, subject=%v", r, a, c)
}

// returns a new unary server interceptors that performs per-request auth
func unaryAuth(authRuntime auth.Runtime) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		newCtx, err := auth0(ctx, authRuntime)
		if err != nil {
			return nil, err
		}
		return handler(newCtx, req)
	}
}

// returns a new stream server interceptors that performs per-request auth
func streamAuth(authRuntime auth.Runtime) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		newCtx, err := auth0(stream.Context(), authRuntime)
		if err != nil {
			return err
		}
		ws := wrapServerStream(stream)
		ws.wrappedContext = newCtx
		return handler(srv, ws)
	}
}

// Auth returns unary and stream interceptors
func Auth(authRuntime auth.Runtime) []grpc.ServerOption {

	return []grpc.ServerOption{
		WithUnaryInterceptors(unaryAuth(authRuntime)),
		WithStreamInterceptors(streamAuth(authRuntime)),
	}
}
