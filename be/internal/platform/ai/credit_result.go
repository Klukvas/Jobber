package ai

import "context"

func MarkCreditAdmission(ctx context.Context) {
	if scope, ok := ctx.Value(creditContextKey{}).(*creditScope); ok {
		scope.mu.Lock()
		scope.admitted = true
		scope.mu.Unlock()
	}
}

func CreditRequestAdmitted(ctx context.Context) bool {
	if scope, ok := ctx.Value(creditContextKey{}).(*creditScope); ok {
		scope.mu.Lock()
		defer scope.mu.Unlock()
		return scope.admitted
	}
	return false
}

// MarkCreditResult records an explicitly successful application outcome. Provider
// settlement alone is never evidence of a usable result.
func MarkCreditResult(ctx context.Context) {
	if scope, ok := ctx.Value(creditContextKey{}).(*creditScope); ok {
		scope.mu.Lock()
		scope.resultReady = true
		scope.mu.Unlock()
	}
}

func CreditResultReady(ctx context.Context) bool {
	if scope, ok := ctx.Value(creditContextKey{}).(*creditScope); ok {
		scope.mu.Lock()
		defer scope.mu.Unlock()
		return scope.resultReady
	}
	return false
}

func RejectCreditResult(ctx context.Context) {
	if scope, ok := ctx.Value(creditContextKey{}).(*creditScope); ok {
		scope.mu.Lock()
		scope.resultReady = false
		scope.mu.Unlock()
	}
}
