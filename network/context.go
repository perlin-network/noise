package network

import (
	"context"
)

type (
	signMessageCtxKeyType string
)

const (
	signMessageCtxKey signMessageCtxKeyType = "signMessage"
)

// WithSignMessage sets whether the request should be signed
func WithSignMessage(ctx context.Context, sign bool) context.Context {
	return context.WithValue(ctx, signMessageCtxKey, sign)
}

// GetSignMessage returns whether the request should be signed
func GetSignMessage(ctx context.Context) bool {
	sign, ok := ctx.Value(signMessageCtxKey).(bool)
	if !ok {
		return false
	}
	return sign
}
