package app

import "context"

type agentRequestIDKey struct{}

func WithAgentRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, agentRequestIDKey{}, id)
}
func agentRequestID(ctx context.Context) string {
	value, _ := ctx.Value(agentRequestIDKey{}).(string)
	return value
}
