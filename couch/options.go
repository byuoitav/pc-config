package couch

import "github.com/go-kivik/couchdb/v3"

const (
	_defaultUIConfigDB  = "ui-configuration"
	_defaultPCMappingDB = "pc-mapping"
)

type options struct {
	authFunc    interface{}
	uiConfigDB  string
	pcMappingDB string
}

// Option configures how we create the DataService.
type Option interface {
	apply(*options)
}

type optionFunc func(*options)

func (f optionFunc) apply(o *options) {
	f(o)
}

func WithBasicAuth(username, password string) Option {
	return optionFunc(func(o *options) {
		o.authFunc = couchdb.BasicAuth(username, password)
	})
}
