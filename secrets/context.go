package secrets

import "context"

type (
	contextServiceKey struct{}
)

// ContextWithService ...
func ContextWithService(ctx context.Context, svc Service) context.Context {
	return context.WithValue(ctx, contextServiceKey{}, svc)
}

// ServiceFromContext ...
func ServiceFromContext(ctx context.Context) Service {
	if v := ctx.Value(contextServiceKey{}); v != nil {
		return v.(Service)
	}
	return nil
}
