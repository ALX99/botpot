package hostprovider

import "context"

// SSH provides SSH hosts
type SSH interface {
	Start(context.Context) error
	Stop(context.Context) error
	GetHost(context.Context) (IP string, id string, err error)
	StopHost(ctx context.Context, id string) error
	GetScriptOutput(ctx context.Context, id string) (string, string, error)
}
