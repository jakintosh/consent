package testing

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"sync"
)

var (
	sharedKey     *ecdsa.PrivateKey
	sharedKeyOnce sync.Once
)

// SharedTestKey returns a cached ECDSA P-256 key for testing.
// Using a shared key avoids the overhead of key generation per test.
func SharedTestKey() *ecdsa.PrivateKey {
	sharedKeyOnce.Do(func() {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			panic("consent/testing: failed to generate key: " + err.Error())
		}
		sharedKey = key
	})
	return sharedKey
}

// GenerateTestKey creates a fresh ECDSA P-256 key.
// Use this when tests need isolated keys (e.g., testing wrong-key scenarios).
func GenerateTestKey() (
	*ecdsa.PrivateKey,
	error,
) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}
