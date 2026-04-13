package middleware

import "go.uber.org/zap"

func ExampleAddLogging() {
	unary, stream := AddLogging(zap.NewNop())
	_, _ = unary, stream
}

func ExampleAddPayloadLogging() {
	unary, stream := AddPayloadLogging(zap.NewNop())
	_, _ = unary, stream
}

func ExampleAddRecovery() {
	unary, stream := AddRecovery()
	_, _ = unary, stream
}
