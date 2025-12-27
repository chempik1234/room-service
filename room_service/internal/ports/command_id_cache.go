package ports

import "context"

// CommandIDShortCache - is the LRU cache for last commands (no-repeat ID)
//
// might be implemented with different storages (e.g. in-memory, redis)
// and mechanisms (e.g. N last saved)
//
// just storing that commandID exists
type CommandIDShortCache interface {
	// Exists - check if commandID exists in DB (if it does, then skip command)
	Exists(ctx context.Context, commandID string) (bool, error)

	// Save - saves commandID, so CommandIDShortCache.Exists returns true
	Save(ctx context.Context, commandID string) error
}
