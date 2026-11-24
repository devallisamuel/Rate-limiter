package limiter

import "time"

type Allower interface {
	Allow() bool
	LastSeen() time.Time
}

var (
	_ Allower = (*Limiter)(nil)
	_ Allower = (*TokenBucket)(nil)
)
