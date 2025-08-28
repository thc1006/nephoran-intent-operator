/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package shared

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Lock represents a distributed lock.
type Lock struct {
	Key        string            `json:"key"`
	Owner      string            `json:"owner"`
	LockType   string            `json:"lockType"`
	AcquiredAt time.Time         `json:"acquiredAt"`
	ExpiresAt  time.Time         `json:"expiresAt"`
	Renewable  bool              `json:"renewable"`
	Metadata   map[string]string `json:"metadata,omitempty"`

	// Internal fields.
	renewalCancel context.CancelFunc
	renewalDone   chan bool
}

// LockManager provides distributed locking capabilities.
type LockManager struct {
	logger logr.Logger
	mutex  sync.RWMutex

	// Active locks.
	locks map[string]*Lock

	// Lock queues for waiting requests.
	waitQueues map[string][]chan bool

	// Configuration.
	defaultTimeout  time.Duration
	renewalInterval time.Duration
	maxRetries      int
	retryBackoff    time.Duration

	// Statistics.
	locksAcquired int64
	locksReleased int64
	lockTimeouts  int64
	lockConflicts int64

	// Background processes.
	cleanupInterval time.Duration
	stopCleanup     chan bool
	cleanupRunning  bool
}

// LockOptions provides options for acquiring locks.
type LockOptions struct {
	Timeout      time.Duration     `json:"timeout,omitempty"`
	LockType     string            `json:"lockType,omitempty"` // "exclusive", "shared"
	Renewable    bool              `json:"renewable,omitempty"`
	Owner        string            `json:"owner,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	RetryCount   int               `json:"retryCount,omitempty"`
	RetryBackoff time.Duration     `json:"retryBackoff,omitempty"`
}

// DefaultLockOptions returns default lock options.
func DefaultLockOptions() *LockOptions {
	return &LockOptions{
		Timeout:      30 * time.Second,
		LockType:     LockTypeExclusive,
		Renewable:    true,
		RetryCount:   3,
		RetryBackoff: 1 * time.Second,
	}
}

// NewLockManager creates a new lock manager.
func NewLockManager(defaultTimeout, retryInterval time.Duration, maxRetries int) *LockManager {
	lm := &LockManager{
		logger:          ctrl.Log.WithName("lock-manager"),
		locks:           make(map[string]*Lock),
		waitQueues:      make(map[string][]chan bool),
		defaultTimeout:  defaultTimeout,
		renewalInterval: retryInterval,
		maxRetries:      maxRetries,
		retryBackoff:    1 * time.Second,
		cleanupInterval: 1 * time.Minute,
		stopCleanup:     make(chan bool),
	}

	// Start background cleanup.
	go lm.runCleanup()

	return lm
}

// AcquireLock acquires a lock with default options.
func (lm *LockManager) AcquireLock(ctx context.Context, key string) error {
	opts := DefaultLockOptions()
	opts.Owner = "default"
	return lm.AcquireLockWithOptions(ctx, key, opts)
}

// AcquireLockWithOptions acquires a lock with specified options.
func (lm *LockManager) AcquireLockWithOptions(ctx context.Context, key string, opts *LockOptions) error {
	if opts == nil {
		opts = DefaultLockOptions()
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = lm.defaultTimeout
	}

	// Create context with timeout.
	lockCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Try to acquire lock with retries.
	for attempt := 0; attempt <= opts.RetryCount; attempt++ {
		if attempt > 0 {
			// Wait before retry.
			select {
			case <-time.After(opts.RetryBackoff):
			case <-lockCtx.Done():
				return fmt.Errorf("lock acquisition cancelled during retry backoff")
			}
		}

		acquired, err := lm.tryAcquireLock(lockCtx, key, opts)
		if err != nil {
			lm.logger.Error(err, "Failed to acquire lock", "key", key, "attempt", attempt+1)
			if attempt == opts.RetryCount {
				return fmt.Errorf("failed to acquire lock after %d attempts: %w", opts.RetryCount+1, err)
			}
			continue
		}

		if acquired {
			lm.locksAcquired++
			return nil
		}

		// Lock is held by someone else, wait or retry.
		if attempt < opts.RetryCount {
			if !lm.waitForLockRelease(lockCtx, key) {
				lm.lockTimeouts++
				return fmt.Errorf("timeout waiting for lock: %s", key)
			}
		}
	}

	lm.lockTimeouts++
	return fmt.Errorf("failed to acquire lock %s after %d attempts", key, opts.RetryCount+1)
}

// ReleaseLock releases a lock.
func (lm *LockManager) ReleaseLock(key string) error {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	lock, exists := lm.locks[key]
	if !exists {
		return fmt.Errorf("lock not found: %s", key)
	}

	// Cancel renewal if active.
	if lock.renewalCancel != nil {
		lock.renewalCancel()
	}

	// Remove lock.
	delete(lm.locks, key)
	lm.locksReleased++

	// Notify waiting goroutines.
	lm.notifyWaiters(key)

	lm.logger.V(1).Info("Lock released", "key", key, "owner", lock.Owner)

	return nil
}

// IsLocked checks if a key is locked.
func (lm *LockManager) IsLocked(key string) bool {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()

	lock, exists := lm.locks[key]
	if !exists {
		return false
	}

	// Check if lock has expired.
	if time.Now().After(lock.ExpiresAt) {
		return false
	}

	return true
}

// GetLock retrieves lock information.
func (lm *LockManager) GetLock(key string) (*Lock, bool) {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()

	lock, exists := lm.locks[key]
	if !exists {
		return nil, false
	}

	// Check if lock has expired.
	if time.Now().After(lock.ExpiresAt) {
		return nil, false
	}

	return lock, true
}

// ActiveLocks returns the number of active locks.
func (lm *LockManager) ActiveLocks() int {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()

	return len(lm.locks)
}

// ListLocks returns all active locks.
func (lm *LockManager) ListLocks() []*Lock {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()

	locks := make([]*Lock, 0, len(lm.locks))
	now := time.Now()

	for _, lock := range lm.locks {
		if now.Before(lock.ExpiresAt) {
			locks = append(locks, lock)
		}
	}

	return locks
}

// RenewLock renews a lock's expiration time.
func (lm *LockManager) RenewLock(key string, duration time.Duration) error {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	lock, exists := lm.locks[key]
	if !exists {
		return fmt.Errorf("lock not found: %s", key)
	}

	if !lock.Renewable {
		return fmt.Errorf("lock is not renewable: %s", key)
	}

	// Check if lock has expired.
	if time.Now().After(lock.ExpiresAt) {
		return fmt.Errorf("lock has expired: %s", key)
	}

	// Renew the lock.
	lock.ExpiresAt = time.Now().Add(duration)

	lm.logger.V(1).Info("Lock renewed", "key", key, "owner", lock.Owner, "expiresAt", lock.ExpiresAt)

	return nil
}

// GetStats returns lock manager statistics.
func (lm *LockManager) GetStats() *LockStats {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()

	return &LockStats{
		ActiveLocks:   int64(len(lm.locks)),
		LocksAcquired: lm.locksAcquired,
		LocksReleased: lm.locksReleased,
		LockTimeouts:  lm.lockTimeouts,
		LockConflicts: lm.lockConflicts,
	}
}

// Stop stops the lock manager.
func (lm *LockManager) Stop() {
	if lm.cleanupRunning {
		lm.stopCleanup <- true
		lm.cleanupRunning = false
	}

	// Release all locks.
	lm.mutex.Lock()
	for key := range lm.locks {
		lm.ReleaseLock(key)
	}
	lm.mutex.Unlock()
}

// Internal methods.

func (lm *LockManager) tryAcquireLock(ctx context.Context, key string, opts *LockOptions) (bool, error) {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	existingLock, exists := lm.locks[key]

	// Check if lock exists and is still valid.
	if exists {
		if time.Now().Before(existingLock.ExpiresAt) {
			// Lock is still active.
			if opts.LockType == LockTypeShared && existingLock.LockType == LockTypeShared {
				// Both are shared locks, allow.
				return true, nil
			}
			// Exclusive lock conflict.
			lm.lockConflicts++
			return false, nil
		}
		// Lock has expired, remove it.
		delete(lm.locks, key)
	}

	// Create new lock.
	lock := &Lock{
		Key:        key,
		Owner:      opts.Owner,
		LockType:   opts.LockType,
		AcquiredAt: time.Now(),
		ExpiresAt:  time.Now().Add(opts.Timeout),
		Renewable:  opts.Renewable,
		Metadata:   opts.Metadata,
	}

	// Start auto-renewal if enabled.
	if lock.Renewable {
		renewalCtx, cancel := context.WithCancel(ctx)
		lock.renewalCancel = cancel
		lock.renewalDone = make(chan bool, 1)

		go lm.autoRenewLock(renewalCtx, lock)
	}

	lm.locks[key] = lock

	lm.logger.V(1).Info("Lock acquired", "key", key, "owner", opts.Owner, "type", opts.LockType)

	return true, nil
}

func (lm *LockManager) waitForLockRelease(ctx context.Context, key string) bool {
	lm.mutex.Lock()

	// Add to wait queue.
	waitChan := make(chan bool, 1)
	lm.waitQueues[key] = append(lm.waitQueues[key], waitChan)

	lm.mutex.Unlock()

	// Wait for notification or timeout.
	select {
	case <-waitChan:
		return true
	case <-ctx.Done():
		// Remove from wait queue.
		lm.mutex.Lock()
		lm.removeFromWaitQueue(key, waitChan)
		lm.mutex.Unlock()
		return false
	}
}

func (lm *LockManager) notifyWaiters(key string) {
	waiters, exists := lm.waitQueues[key]
	if !exists {
		return
	}

	// Notify all waiters.
	for _, waitChan := range waiters {
		select {
		case waitChan <- true:
		default:
			// Channel is full or closed.
		}
	}

	// Clear the wait queue.
	delete(lm.waitQueues, key)
}

func (lm *LockManager) removeFromWaitQueue(key string, targetChan chan bool) {
	waiters, exists := lm.waitQueues[key]
	if !exists {
		return
	}

	// Remove the specific channel from the wait queue.
	for i, waitChan := range waiters {
		if waitChan == targetChan {
			lm.waitQueues[key] = append(waiters[:i], waiters[i+1:]...)
			break
		}
	}

	// Clean up empty wait queue.
	if len(lm.waitQueues[key]) == 0 {
		delete(lm.waitQueues, key)
	}
}

func (lm *LockManager) autoRenewLock(ctx context.Context, lock *Lock) {
	ticker := time.NewTicker(lm.renewalInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if lock still exists.
			lm.mutex.RLock()
			_, exists := lm.locks[lock.Key]
			lm.mutex.RUnlock()

			if !exists {
				return
			}

			// Renew the lock.
			renewalDuration := lm.renewalInterval * 3 // Give some buffer
			if err := lm.RenewLock(lock.Key, renewalDuration); err != nil {
				lm.logger.Error(err, "Failed to renew lock", "key", lock.Key)
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

func (lm *LockManager) runCleanup() {
	lm.cleanupRunning = true
	ticker := time.NewTicker(lm.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			lm.cleanupExpiredLocks()
		case <-lm.stopCleanup:
			return
		}
	}
}

func (lm *LockManager) cleanupExpiredLocks() {
	lm.mutex.Lock()
	defer lm.mutex.Unlock()

	now := time.Now()
	expired := make([]string, 0)

	for key, lock := range lm.locks {
		if now.After(lock.ExpiresAt) {
			expired = append(expired, key)
		}
	}

	for _, key := range expired {
		lock := lm.locks[key]

		// Cancel renewal if active.
		if lock.renewalCancel != nil {
			lock.renewalCancel()
		}

		delete(lm.locks, key)
		lm.notifyWaiters(key)

		lm.logger.V(1).Info("Expired lock cleaned up", "key", key, "owner", lock.Owner)
	}
}

// LockStats provides statistics about the lock manager.
type LockStats struct {
	ActiveLocks   int64 `json:"activeLocks"`
	LocksAcquired int64 `json:"locksAcquired"`
	LocksReleased int64 `json:"locksReleased"`
	LockTimeouts  int64 `json:"lockTimeouts"`
	LockConflicts int64 `json:"lockConflicts"`
}

// LockWithContext provides a context-aware lock implementation.
func (lm *LockManager) LockWithContext(ctx context.Context, key string, fn func() error) error {
	if err := lm.AcquireLock(ctx, key); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	defer func() {
		if err := lm.ReleaseLock(key); err != nil {
			lm.logger.Error(err, "Failed to release lock", "key", key)
		}
	}()

	return fn()
}

// TryLockWithTimeout attempts to acquire a lock with a timeout.
func (lm *LockManager) TryLockWithTimeout(ctx context.Context, key string, timeout time.Duration) error {
	opts := DefaultLockOptions()
	opts.Timeout = timeout
	opts.Owner = "timeout-lock"

	return lm.AcquireLockWithOptions(ctx, key, opts)
}
