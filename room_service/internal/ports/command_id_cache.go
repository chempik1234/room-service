package ports

import "github.com/chempik1234/super-danis-library-golang/pkg/pkgports"

// CommandIdShortCache - is the LRU cache for last commands (no-repeat ID)
//
// might be implemented with different storages (e.g. in-memory, redis)
// and mechanisms (e.g. N last saved)
//
// just storing that commandID exists
type CommandIdShortCache pkgports.Cache[Tuple, struct{}]
