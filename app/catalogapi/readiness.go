package catalogapi

import "context"

type readinessPinger interface {
	Ping(context.Context) error
}
