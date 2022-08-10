package fab

import (
	"context"
	"sync"
)

// Target is the interface that Fab targets must implement.
type Target interface {
	Run(context.Context) error
	Once() *sync.Once
}
