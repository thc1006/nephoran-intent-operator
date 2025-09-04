package auth

import (
	"context"
	"sync"
	"time"
)

// MemoryTokenStore provides an in-memory implementation of TokenStore for testing
type MemoryTokenStore struct {
	tokens map[string]*TokenInfo
	mutex  sync.RWMutex
}

// NewMemoryTokenStore creates a new in-memory token store
func NewMemoryTokenStore() *MemoryTokenStore {
	return &MemoryTokenStore{
		tokens: make(map[string]*TokenInfo),
	}
}

// StoreToken stores a token with expiration
func (s *MemoryTokenStore) StoreToken(ctx context.Context, tokenID string, token *TokenInfo) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Create a copy to avoid mutations
	tokenCopy := *token
	s.tokens[tokenID] = &tokenCopy

	return nil
}

// GetToken retrieves token info
func (s *MemoryTokenStore) GetToken(ctx context.Context, tokenID string) (*TokenInfo, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	token, exists := s.tokens[tokenID]
	if !exists {
		return nil, nil
	}

	// Return a copy to avoid mutations
	tokenCopy := *token
	return &tokenCopy, nil
}

// UpdateToken updates token info
func (s *MemoryTokenStore) UpdateToken(ctx context.Context, tokenID string, token *TokenInfo) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.tokens[tokenID]; !exists {
		return nil // Token doesn't exist
	}

	// Create a copy to avoid mutations
	tokenCopy := *token
	s.tokens[tokenID] = &tokenCopy

	return nil
}

// DeleteToken deletes a token
func (s *MemoryTokenStore) DeleteToken(ctx context.Context, tokenID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.tokens, tokenID)
	return nil
}

// ListUserTokens lists tokens for a user
func (s *MemoryTokenStore) ListUserTokens(ctx context.Context, userID string) ([]*TokenInfo, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var userTokens []*TokenInfo
	for _, token := range s.tokens {
		if token.UserID == userID {
			tokenCopy := *token
			userTokens = append(userTokens, &tokenCopy)
		}
	}

	return userTokens, nil
}

// CleanupExpired removes expired tokens
func (s *MemoryTokenStore) CleanupExpired(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	for tokenID, token := range s.tokens {
		if now.After(token.ExpiresAt) {
			delete(s.tokens, tokenID)
		}
	}

	return nil
}

// MemoryTokenBlacklist provides an in-memory implementation of TokenBlacklist for testing
type MemoryTokenBlacklist struct {
	blacklisted map[string]time.Time
	mutex       sync.RWMutex
}

// NewMemoryTokenBlacklist creates a new in-memory token blacklist
func NewMemoryTokenBlacklist() *MemoryTokenBlacklist {
	return &MemoryTokenBlacklist{
		blacklisted: make(map[string]time.Time),
	}
}

// BlacklistToken adds a token to the blacklist
func (b *MemoryTokenBlacklist) BlacklistToken(ctx context.Context, tokenID string, expiresAt time.Time) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.blacklisted[tokenID] = expiresAt
	return nil
}

// IsTokenBlacklisted checks if a token is blacklisted
func (b *MemoryTokenBlacklist) IsTokenBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	_, exists := b.blacklisted[tokenID]
	return exists, nil
}

// CleanupExpired removes expired blacklist entries
func (b *MemoryTokenBlacklist) CleanupExpired(ctx context.Context) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	now := time.Now()
	for tokenID, expiresAt := range b.blacklisted {
		if now.After(expiresAt) {
			delete(b.blacklisted, tokenID)
		}
	}

	return nil
}
