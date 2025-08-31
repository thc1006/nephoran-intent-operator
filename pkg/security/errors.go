package security

import "errors"

var (
	// ErrSecretNotFound is returned when a secret is not found
	ErrSecretNotFound = errors.New("secret not found")
	
	// ErrKeyNotFound holds errkeynotfound value.
	ErrKeyNotFound = errors.New("key not found")
)