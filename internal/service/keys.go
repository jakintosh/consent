package service

import (
	"errors"
	"fmt"

	"git.sr.ht/~jakintosh/command-go/pkg/keys"
)

var (
	PermissionRead = keys.Permission{
		Key:         "read",
		Display:     "Read",
		Description: "Read-only API access",
	}
	PermissionWrite = keys.Permission{
		Key:         "write",
		Display:     "Write",
		Description: "Mutating API access",
	}
	PermissionAdmin = keys.Permission{
		Key:         "admin",
		Display:     "Admin",
		Description: "Administrative access",
	}
)

func AllKeyPermissions() []keys.Permission {
	return []keys.Permission{
		PermissionRead,
		PermissionWrite,
		PermissionAdmin,
	}
}

func AllKeyPermissionRefs() []*keys.Permission {
	return []*keys.Permission{
		&PermissionRead,
		&PermissionWrite,
		&PermissionAdmin,
	}
}

func NewKeysService(store keys.Store) (*keys.Service, error) {
	if store == nil {
		return nil, fmt.Errorf("service: keys store required")
	}

	return keys.New(keys.Options{
		Store:       store,
		Permissions: AllKeyPermissions(),
	})
}

func InitKeys(store keys.Store, bootstrapToken string) error {
	if bootstrapToken == "" {
		return fmt.Errorf("service: bootstrap token required")
	}

	keysSvc, err := NewKeysService(store)
	if err != nil {
		return err
	}

	err = keysSvc.Init(bootstrapToken, AllKeyPermissionRefs()...)
	if err != nil && !errors.Is(err, keys.ErrAlreadyInitialized) {
		return fmt.Errorf("service: initialize keys: %w", err)
	}

	return nil
}
