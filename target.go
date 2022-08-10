package fab

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

// Target is the interface that Fab targets must implement.
type Target interface {
	// Run invokes the target's logic.
	Run(context.Context) error

	// ID is a unique ID for the target.
	ID() string
}

func RandID() string {
	var buf [16]byte
	rand.Read(buf[:])
	return hex.EncodeToString(buf[:])
}
