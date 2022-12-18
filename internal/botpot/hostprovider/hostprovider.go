package hostprovider

import "context"

// SSH provides SSH hosts
type SSH interface {
	Start(context.Context) error
	Stop(context.Context) error
	GetHost(context.Context) (IP string, ID string, err error)
	StopHost(ctx context.Context, ID string) error
	GetScriptOutput(ctx context.Context, ID string) (string, string, error)
}
