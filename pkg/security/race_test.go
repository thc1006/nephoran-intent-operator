package security

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/testing/racetest"
)

// TestCertificateManagerRaceConditions tests concurrent certificate operations
func TestCertificateManagerRaceConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping race test in short mode")
	}

	runner := racetest.NewRunner(t, &racetest.RaceTestConfig{
		Goroutines: 30,
		Iterations: 50,
		Timeout:    10 * time.Second,
	})

	certManager := &certificateManager{
		mu:          sync.RWMutex{},
		certs:       make(map[string]*x509.Certificate),
		keys:        make(map[string]*rsa.PrivateKey),
		rotations:   atomic.Int64{},
		validations: atomic.Int64{},
	}

	runner.RunConcurrent(func(id int) error {
		certID := fmt.Sprintf("cert-%d", id%10)

		// Certificate rotation check
		certManager.mu.RLock()
		cert, exists := certManager.certs[certID]
		certManager.mu.RUnlock()

		if !exists || (cert != nil && time.Until(cert.NotAfter) < 24*time.Hour) {
			// Rotate certificate
			certManager.mu.Lock()
			// Double-check after acquiring write lock
			if cert, exists := certManager.certs[certID]; !exists ||
				(cert != nil && time.Until(cert.NotAfter) < 24*time.Hour) {
				// Generate new certificate (simplified)
				newCert := &x509.Certificate{
					SerialNumber: big.NewInt(int64(id)),
					NotAfter:     time.Now().Add(365 * 24 * time.Hour),
				}
				certManager.certs[certID] = newCert
				certManager.rotations.Add(1)
			}
			certManager.mu.Unlock()
		}

		// Validate certificate
		certManager.mu.RLock()
		if cert := certManager.certs[certID]; cert != nil {
			// Validation logic
			certManager.validations.Add(1)
		}
		certManager.mu.RUnlock()

		return nil
	})

	t.Logf("Rotations: %d, Validations: %d, Total certs: %d",
		certManager.rotations.Load(), certManager.validations.Load(),
		len(certManager.certs))
}

// TestTLSSessionCacheRace tests TLS session cache concurrent access
func TestTLSSessionCacheRace(t *testing.T) {
	var wg sync.WaitGroup
	defer wg.Wait()

	sessionCache := &tlsSessionCache{
		sessions:    &sync.Map{},
		maxSessions: 100,
		evictions:   atomic.Int64{},
		hits:        atomic.Int64{},
		misses:      atomic.Int64{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Concurrent session operations
	for i := 0; i < 50; i++ {
		sessionID := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				select {
				case <-ctx.Done():
					return
				default:
					key := fmt.Sprintf("session-%d", sessionID%20)

					// Try to get session
					if session, ok := sessionCache.sessions.Load(key); ok {
						sessionCache.hits.Add(1)
						// Update last access time
						if s, ok := session.(*tlsSession); ok {
							atomic.StoreInt64(&s.lastAccess, time.Now().UnixNano())
						}
					} else {
						sessionCache.misses.Add(1)
						// Create new session
						newSession := &tlsSession{
							id:         key,
							created:    time.Now().UnixNano(),
							lastAccess: time.Now().UnixNano(),
						}

						// Check if we need to evict
						count := countSyncMapEntries(sessionCache.sessions)
						if count >= sessionCache.maxSessions {
							// Find and evict oldest session
							var oldestKey string
							var oldestTime int64 = time.Now().UnixNano()

							sessionCache.sessions.Range(func(k, v interface{}) bool {
								if s, ok := v.(*tlsSession); ok {
									if accessTime := atomic.LoadInt64(&s.lastAccess); accessTime < oldestTime {
										oldestTime = accessTime
										oldestKey = k.(string)
									}
								}
								return true
							})

							if oldestKey != "" {
								sessionCache.sessions.Delete(oldestKey)
								sessionCache.evictions.Add(1)
							}
						}

						sessionCache.sessions.Store(key, newSession)
					}
				}
			}
		}()
	}

	group.Wait()
	t.Logf("Hits: %d, Misses: %d, Evictions: %d, Current sessions: %d",
		sessionCache.hits.Load(), sessionCache.misses.Load(),
		sessionCache.evictions.Load(), countSyncMapEntries(sessionCache.sessions))
}

// TestRBACAuthorizationRace tests RBAC authorization under concurrent load
func TestRBACAuthorizationRace(t *testing.T) {
	runner := racetest.NewRunner(t, &racetest.RaceTestConfig{
		Goroutines: 100,
		Iterations: 100,
		Timeout:    10 * time.Second,
	})

	rbac := &rbacAuthorizer{
		mu:          sync.RWMutex{},
		roles:       make(map[string]*role),
		userRoles:   make(map[string][]string),
		permissions: &sync.Map{},
		authCount:   atomic.Int64{},
		denyCount:   atomic.Int64{},
	}

	// Initialize some roles
	rbac.mu.Lock()
	rbac.roles["admin"] = &role{
		name:        "admin",
		permissions: []string{"read", "write", "delete"},
	}
	rbac.roles["user"] = &role{
		name:        "user",
		permissions: []string{"read"},
	}
	rbac.userRoles["user1"] = []string{"admin"}
	rbac.userRoles["user2"] = []string{"user"}
	rbac.mu.Unlock()

	runner.RunConcurrent(func(id int) error {
		userID := fmt.Sprintf("user%d", id%3)
		resource := fmt.Sprintf("resource-%d", id%10)
		action := []string{"read", "write", "delete"}[id%3]

		// Check authorization
		rbac.mu.RLock()
		userRoles := rbac.userRoles[userID]
		allowed := false

		for _, roleName := range userRoles {
			if role := rbac.roles[roleName]; role != nil {
				for _, perm := range role.permissions {
					if perm == action {
						allowed = true
						break
					}
				}
			}
			if allowed {
				break
			}
		}
		rbac.mu.RUnlock()

		// Cache permission decision
		cacheKey := fmt.Sprintf("%s:%s:%s", userID, resource, action)
		rbac.permissions.Store(cacheKey, allowed)

		if allowed {
			rbac.authCount.Add(1)
		} else {
			rbac.denyCount.Add(1)
		}

		// Occasionally update roles (simulate dynamic RBAC)
		if id%100 == 0 {
			rbac.mu.Lock()
			newUser := fmt.Sprintf("user%d", id)
			rbac.userRoles[newUser] = []string{"user"}
			rbac.mu.Unlock()
		}

		return nil
	})

	t.Logf("Authorizations: %d, Denials: %d, Cached permissions: %d",
		rbac.authCount.Load(), rbac.denyCount.Load(),
		countSyncMapEntries(rbac.permissions))
}

// TestSecretRotationRace tests secret rotation with concurrent access
func TestSecretRotationRace(t *testing.T) {
	atomicTest := racetest.NewAtomicRaceTest(t)

	secretManager := &secretRotationManager{
		secrets:      &sync.Map{},
		version:      atomic.Int64{},
		rotationLock: sync.Mutex{},
	}

	// Test atomic version updates
	atomicTest.TestCompareAndSwap(&secretManager.version)

	runner := racetest.NewRunner(t, racetest.DefaultConfig())

	var reads, rotations atomic.Int64

	runner.RunConcurrent(func(id int) error {
		secretID := fmt.Sprintf("secret-%d", id%5)

		if id%20 == 0 {
			// Rotate secret
			secretManager.rotationLock.Lock()
			oldVersion := secretManager.version.Load()

			// Generate new secret
			newSecret := make([]byte, 32)
			rand.Read(newSecret)

			secretManager.secrets.Store(secretID, &secret{
				value:   newSecret,
				version: oldVersion + 1,
			})

			secretManager.version.Add(1)
			rotations.Add(1)
			secretManager.rotationLock.Unlock()
		} else {
			// Read secret
			if val, ok := secretManager.secrets.Load(secretID); ok {
				if s, ok := val.(*secret); ok {
					// Use secret
					_ = s.value
					reads.Add(1)
				}
			}
		}

		return nil
	})

	t.Logf("Reads: %d, Rotations: %d, Current version: %d",
		reads.Load(), rotations.Load(), secretManager.version.Load())
}

// TestCryptoKeyPoolRace tests cryptographic key pool management
func TestCryptoKeyPoolRace(t *testing.T) {
	mutexTest := racetest.NewMutexRaceTest(t)

	keyPool := &cryptoKeyPool{
		mu:        sync.RWMutex{},
		keys:      make(map[string]*rsa.PrivateKey),
		inUse:     make(map[string]bool),
		generated: atomic.Int64{},
	}

	mutexTest.TestCriticalSection(&keyPool.mu, &keyPool.keys)

	runner := racetest.NewRunner(t, &racetest.RaceTestConfig{
		Goroutines: 20,
		Iterations: 50,
		Timeout:    15 * time.Second, // Key generation is slow
	})

	runner.RunConcurrent(func(id int) error {
		keyID := fmt.Sprintf("key-%d", id%10)

		// Acquire key
		keyPool.mu.Lock()
		if keyPool.inUse[keyID] {
			keyPool.mu.Unlock()
			return nil
		}

		if _, exists := keyPool.keys[keyID]; !exists {
			// Generate new key (simplified for testing)
			key, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				keyPool.mu.Unlock()
				return err
			}
			keyPool.keys[keyID] = key
			keyPool.generated.Add(1)
		}

		keyPool.inUse[keyID] = true
		key := keyPool.keys[keyID]
		keyPool.mu.Unlock()

		// Use key for operation
		_ = key
		time.Sleep(time.Microsecond)

		// Release key
		keyPool.mu.Lock()
		keyPool.inUse[keyID] = false
		keyPool.mu.Unlock()

		return nil
	})

	t.Logf("Generated keys: %d, Total keys: %d",
		keyPool.generated.Load(), len(keyPool.keys))
}

// TestAuditLogRace tests concurrent audit log writes
func TestAuditLogRace(t *testing.T) {
	channelTest := racetest.NewChannelRaceTest(t)

	auditChan := make(chan *auditEntry, 100)
	channelTest.TestConcurrentSendReceive(auditChan, 10, 5)

	// Test audit log writer
	auditLog := &auditLogger{
		mu:      sync.Mutex{},
		buffer:  make([]*auditEntry, 0, 1000),
		written: atomic.Int64{},
		dropped: atomic.Int64{},
		logChan: make(chan *auditEntry, 100),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Background writer
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case entry := <-auditLog.logChan:
				auditLog.mu.Lock()
				if len(auditLog.buffer) < cap(auditLog.buffer) {
					auditLog.buffer = append(auditLog.buffer, entry)
					auditLog.written.Add(1)
				} else {
					auditLog.dropped.Add(1)
				}
				auditLog.mu.Unlock()
			}
		}
	}()

	runner := racetest.NewRunner(t, &racetest.RaceTestConfig{
		Goroutines: 50,
		Iterations: 100,
		Timeout:    5 * time.Second,
	})

	runner.RunConcurrent(func(id int) error {
		entry := &auditEntry{
			timestamp: time.Now(),
			user:      fmt.Sprintf("user-%d", id),
			action:    fmt.Sprintf("action-%d", id%10),
			resource:  fmt.Sprintf("resource-%d", id%20),
		}

		select {
		case auditLog.logChan <- entry:
			// Success
		default:
			auditLog.dropped.Add(1)
		}

		return nil
	})

	cancel()
	time.Sleep(100 * time.Millisecond) // Let writer finish

	t.Logf("Written: %d, Dropped: %d, Buffer size: %d",
		auditLog.written.Load(), auditLog.dropped.Load(),
		len(auditLog.buffer))
}

// TestDeadlockDetectionInSecurity tests for potential deadlocks
func TestDeadlockDetectionInSecurity(t *testing.T) {
	type multiLockSystem struct {
		lock1 sync.Mutex
		lock2 sync.Mutex
		lock3 sync.Mutex
		ops   atomic.Int64
	}

	system := &multiLockSystem{}

	runner := racetest.NewRunner(t, &racetest.RaceTestConfig{
		Goroutines:        10,
		Iterations:        100,
		Timeout:           5 * time.Second,
		DeadlockDetection: true,
	})

	runner.RunConcurrent(func(id int) error {
		// Acquire locks in consistent order to avoid deadlock
		if id%2 == 0 {
			system.lock1.Lock()
			system.lock2.Lock()
			system.lock3.Lock()
		} else {
			// Same order!
			system.lock1.Lock()
			system.lock2.Lock()
			system.lock3.Lock()
		}

		system.ops.Add(1)

		// Release in reverse order
		system.lock3.Unlock()
		system.lock2.Unlock()
		system.lock1.Unlock()

		return nil
	})

	t.Logf("Operations completed: %d", system.ops.Load())
}

// BenchmarkSecurityConcurrentOperations benchmarks security operations
func BenchmarkSecurityConcurrentOperations(b *testing.B) {
	racetest.BenchmarkRaceConditions(b, func() {
		// Simulate certificate validation
		cert := &x509.Certificate{
			SerialNumber: big.NewInt(1),
			NotAfter:     time.Now().Add(time.Hour),
		}

		if time.Now().Before(cert.NotAfter) {
			// Valid
		}
	})
}

// Helper types
type certificateManager struct {
	mu          sync.RWMutex
	certs       map[string]*x509.Certificate
	keys        map[string]*rsa.PrivateKey
	rotations   atomic.Int64
	validations atomic.Int64
}

type tlsSessionCache struct {
	sessions    *sync.Map
	maxSessions int
	evictions   atomic.Int64
	hits        atomic.Int64
	misses      atomic.Int64
}

type tlsSession struct {
	id         string
	created    int64
	lastAccess int64
}

type rbacAuthorizer struct {
	mu          sync.RWMutex
	roles       map[string]*role
	userRoles   map[string][]string
	permissions *sync.Map
	authCount   atomic.Int64
	denyCount   atomic.Int64
}

type role struct {
	name        string
	permissions []string
}

type secretRotationManager struct {
	secrets      *sync.Map
	version      atomic.Int64
	rotationLock sync.Mutex
}

type secret struct {
	value   []byte
	version int64
}

type cryptoKeyPool struct {
	mu        sync.RWMutex
	keys      map[string]*rsa.PrivateKey
	inUse     map[string]bool
	generated atomic.Int64
}

type auditLogger struct {
	mu      sync.Mutex
	buffer  []*auditEntry
	written atomic.Int64
	dropped atomic.Int64
	logChan chan *auditEntry
}

type auditEntry struct {
	timestamp time.Time
	user      string
	action    string
	resource  string
}

func countSyncMapEntries(m *sync.Map) int {
	count := 0
	m.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	return count
}
