package validator

import (
	"context"
	"google.golang.org/grpc"
)

// UnaryServerInterceptor returns a new unary server interceptor that validates incoming messages.
//
// Invalid messages will be rejected with `InvalidArgument` before reaching any userspace handlers.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if v, ok := req.(allValidator); ok {
			if err := v.ValidateAll(); err != nil {
				return nil, &validationWrapperError{
					error: err,
				}
			}
		} else if v, ok := req.(singleValidator); ok {
			if err := v.Validate(); err != nil {
				return nil, &validationWrapperError{
					error: err,
				}
			}
		}
		return handler(ctx, req)
	}
}

// StreamServerInterceptor returns a new streaming server interceptor that validates incoming messages.
//
// The stage at which invalid messages will be rejected with `InvalidArgument` varies based on the
// type of the RPC. For `ServerStream` (1:m) requests, it will happen before reaching any userspace
// handlers. For `ClientStream` (n:1) or `BidiStream` (n:m) RPCs, the messages will be rejected on
// calls to `stream.Recv()`.
func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		wrapper := &recvWrapper{stream}
		return handler(srv, wrapper)
	}
}

type recvWrapper struct {
	grpc.ServerStream
}

func (s *recvWrapper) RecvMsg(m interface{}) error {
	if err := s.ServerStream.RecvMsg(m); err != nil {
		return err
	}
	if v, ok := m.(allValidator); ok {
		if err := v.ValidateAll(); err != nil {
			return &validationWrapperError{
				error: err,
			}
		}
	} else if v, ok := m.(singleValidator); ok {
		if err := v.Validate(); err != nil {
			return &validationWrapperError{
				error: err,
			}
		}
	}
	return nil
}
