package fab

import (
	"context"
	"sync"
)

type Target interface {
	Run(context.Context) error
	Once() *sync.Once
}
