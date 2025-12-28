package models

import "github.com/chempik1234/super-danis-library-golang/v2/pkg/types"

// User is the "user_full" model
type User struct {
	Metadata map[string]string
	ID       types.NotEmptyText
	Name     types.NotEmptyText
}
