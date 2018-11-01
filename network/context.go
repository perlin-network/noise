package network

import (
	"context"
	"time"
)

type (
	signMessageCtxKeyType string
)

const (
	ctxKeyStrongSignature         signMessageCtxKeyType = "stronglySign"
	ctxKeyWeakSignature           signMessageCtxKeyType = "weaklySign"
	ctxKeyWeakSignatureExpiration signMessageCtxKeyType = "weaklySignExpiration"
)

// WithStrongSignature sets whether the request should signed with a strong signature
func WithStrongSignature(ctx context.Context, sign bool) context.Context {
	return context.WithValue(ctx, ctxKeyStrongSignature, sign)
}

// GetStrongSignature returns whether the request should be signed with a strong signature
func GetStrongSignature(ctx context.Context) bool {
	sign, ok := ctx.Value(ctxKeyStrongSignature).(bool)
	if !ok {
		return false
	}
	return sign
}

// WithWeakSignature sets whether the request should be signed with a weak signature and how long the signature is valid
func WithWeakSignature(ctx context.Context, sign bool, expiration *time.Time) context.Context {
	ctx = context.WithValue(ctx, ctxKeyWeakSignature, sign)
	if expiration != nil {
		ctx = context.WithValue(ctx, ctxKeyWeakSignatureExpiration, expiration)
	}
	return ctx
}

// GetWeakSignature returns whether the request should be signed with a weak signature and the expiration if specified
func GetWeakSignature(ctx context.Context) (bool, *time.Time) {
	sign, ok := ctx.Value(ctxKeyWeakSignature).(bool)
	if !ok {
		return false, nil
	}
	expiration, ok := ctx.Value(ctxKeyWeakSignatureExpiration).(*time.Time)
	if !ok {
		return sign, nil
	}
	return sign, expiration
}
