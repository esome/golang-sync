// Package channels provides a set of utilities for working with channels.

package channels

import "fmt"

var (
	// ErrBroadcastWithNoReceivers is returned, when Broadcast is called with no receiver channels
	ErrBroadcastWithNoReceivers = fmt.Errorf("broadcast called with no receiver channels")
	// ErrInvalidChunkSize is returned, when ChunkAndDo is called with chunk size of 0
	ErrInvalidChunkSize = fmt.Errorf("parameter size must be greater than 0")
)
