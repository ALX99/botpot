package hostprovider

// SSH provides SSH hosts
type SSH interface {
	Start() error
	Stop() error
	GetHost() (IP string, ID string, err error)
	StopHost(ID string) error
}
