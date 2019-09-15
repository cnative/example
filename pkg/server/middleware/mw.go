package middleware

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type wrappedServerStream struct {
	grpc.ServerStream
	wrappedContext context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.wrappedContext
}

func wrapServerStream(stream grpc.ServerStream) *wrappedServerStream {
	if existing, ok := stream.(*wrappedServerStream); ok {
		return existing
	}
	return &wrappedServerStream{ServerStream: stream, wrappedContext: stream.Context()}
}

// Used if no interceptors are specified while chaining
func defaultUnaryInterceptor(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return handler(ctx, req)
}

func chainingUnaryInterceptor(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	n := len(interceptors)
	switch n {
	case 0:
		return defaultUnaryInterceptor
	case 1:
		return interceptors[0]
	default: // n > 1
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

			cur := 0
			var next grpc.UnaryHandler
			next = func(currentCtx context.Context, currentReq interface{}) (interface{}, error) {
				if cur == n-1 {
					return handler(currentCtx, currentReq)
				}
				cur++
				resp, err := interceptors[cur](currentCtx, currentReq, info, next)
				cur--
				return resp, err
			}

			return interceptors[0](ctx, req, info, next)
		}
	}
}

// Used if no interceptors are specified while chaining
func defaultStreamInterceptor(srv interface{}, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return handler(srv, stream)
}

func chainingStreamInterceptor(interceptors ...grpc.StreamServerInterceptor) grpc.StreamServerInterceptor {
	n := len(interceptors)
	switch n {
	case 0:
		return defaultStreamInterceptor
	case 1:
		return interceptors[0]
	default: // n > 1
		return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {

			cur := 0
			var next grpc.StreamHandler
			next = func(currentSrv interface{}, currentStream grpc.ServerStream) error {
				if cur == n-1 {
					return handler(currentSrv, currentStream)
				}
				cur++
				err := interceptors[cur](currentSrv, currentStream, info, next)
				cur--
				return err
			}

			return interceptors[0](srv, stream, info, next)
		}
	}
}

// WithUnaryInterceptors is a wrapper middleware that chains a set of interceptors in the specified order
func WithUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor) grpc.ServerOption {
	return grpc.UnaryInterceptor(chainingUnaryInterceptor(interceptors...))
}

// WithStreamInterceptors is a wrapper middleware that chains a set of interceptors in the specified order
func WithStreamInterceptors(interceptors ...grpc.StreamServerInterceptor) grpc.ServerOption {
	return grpc.StreamInterceptor(chainingStreamInterceptor(interceptors...))
}
