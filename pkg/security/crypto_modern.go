// Package security provides modern cryptographic operations with Go 1.24+ features.

package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
)

// CryptoModern provides advanced cryptographic operations.

type CryptoModern struct {
	// Entropy pool for enhanced randomness
	entropyPool *EntropyPool

	// Key derivation manager
	keyDerivation *KeyDerivationManager

	// Encryption cache for performance
	encryptionCache *EncryptionCache

	// Key derivation parameters
	kdfParams *KDFParams

	// Encryption contexts
	contexts map[string]*EncryptionContext

	// Post-quantum preparation
	pqReady bool

	mu sync.RWMutex
}

// EntropyPool manages entropy for secure random generation.

type EntropyPool struct {
	pool []byte

	poolSize int

	lastSeed time.Time

	mu sync.Mutex

	rng io.Reader
}

// KDFParams contains key derivation function parameters.

type KDFParams struct {
	// Argon2 parameters.

	Argon2Time uint32

	Argon2Memory uint32

	Argon2Threads uint8

	Argon2KeyLen uint32

	// PBKDF2 parameters.

	PBKDF2Iterations int

	// Scrypt parameters.

	ScryptN int

	ScryptR int

	ScryptP int

	// HKDF info.

	HKDFInfo []byte
}

// EncryptionContext maintains encryption state.

type EncryptionContext struct {
	Algorithm string

	Key []byte

	Nonce []byte

	AAD []byte // Additional Authenticated Data

	Created time.Time
}

// EncryptedData represents encrypted data with metadata.

type EncryptedData struct {
	Algorithm string `json:"algorithm"`

	Ciphertext []byte `json:"ciphertext"`

	Nonce []byte `json:"nonce"`

	AAD []byte `json:"aad,omitempty"`

	Salt []byte `json:"salt,omitempty"`

	Tag []byte `json:"tag,omitempty"`

	Created time.Time `json:"created"`

	Version int `json:"version"`
}

// Ed25519KeyPair represents an Ed25519 key pair.

type Ed25519KeyPair struct {
	PublicKey ed25519.PublicKey

	PrivateKey ed25519.PrivateKey

	Created time.Time

	ID string
}

// NewCryptoModern creates a new modern crypto instance.

func NewCryptoModern() *CryptoModern {
	return &CryptoModern{
		entropyPool:     NewEntropyPool(4096),
		keyDerivation:   NewKeyDerivationManager(),
		encryptionCache: NewEncryptionCache(),
		kdfParams: &KDFParams{
			// Argon2id parameters (recommended for password hashing)
			Argon2Time:    3,
			Argon2Memory:  64 * 1024, // 64 MB
			Argon2Threads: 4,
			Argon2KeyLen:  32,
			// PBKDF2 parameters
			PBKDF2Iterations: 600000, // NIST recommended minimum
			// Scrypt parameters
			ScryptN: 32768,
			ScryptR: 8,
			ScryptP: 1,
			// HKDF
			HKDFInfo: []byte("nephoran-intent-operator"),
		},
		contexts: make(map[string]*EncryptionContext),
		pqReady:  false,
	}
}

// NewEntropyPool creates a new entropy pool.

func NewEntropyPool(size int) *EntropyPool {
	pool := &EntropyPool{
		poolSize: size,

		pool: make([]byte, size),

		rng: rand.Reader,
	}

	pool.reseed()

	return pool
}

// reseed refreshes the entropy pool.

func (p *EntropyPool) reseed() error {
	p.mu.Lock()

	defer p.mu.Unlock()

	if _, err := io.ReadFull(p.rng, p.pool); err != nil {
		return fmt.Errorf("failed to reseed entropy pool: %w", err)
	}

	p.lastSeed = time.Now()

	return nil
}

// Read implements io.Reader for entropy pool.

func (p *EntropyPool) Read(b []byte) (int, error) {
	p.mu.Lock()

	defer p.mu.Unlock()

	// Reseed if pool is older than 5 minutes.

	if time.Since(p.lastSeed) > 5*time.Minute {
		if err := p.reseed(); err != nil {
			return 0, err
		}
	}

	// Mix pool entropy with system randomness.

	systemRandom := make([]byte, len(b))

	if _, err := io.ReadFull(p.rng, systemRandom); err != nil {
		return 0, err
	}

	for i := range b {
		b[i] = p.pool[i%p.poolSize] ^ systemRandom[i]
	}

	// Stir the pool.

	for i := range p.poolSize {
		p.pool[i] ^= byte(i)
	}

	return len(b), nil
}

// EncryptAESGCM performs AES-GCM encryption with additional authenticated data.

func (c *CryptoModern) EncryptAESGCM(plaintext, key, aad []byte) (*EncryptedData, error) {
	// Validate key size.

	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, errors.New("invalid AES key size")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce.

	nonce := make([]byte, gcm.NonceSize())

	if _, err := c.entropyPool.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt with AAD.

	ciphertext := gcm.Seal(nil, nonce, plaintext, aad)

	return &EncryptedData{
		Algorithm: "AES-GCM",

		Ciphertext: ciphertext,

		Nonce: nonce,

		AAD: aad,

		Created: time.Now(),

		Version: 1,
	}, nil
}

// DecryptAESGCM performs AES-GCM decryption with additional authenticated data.

func (c *CryptoModern) DecryptAESGCM(data *EncryptedData, key []byte) ([]byte, error) {
	if data.Algorithm != "AES-GCM" {
		return nil, fmt.Errorf("invalid algorithm: expected AES-GCM, got %s", data.Algorithm)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, data.Nonce, data.Ciphertext, data.AAD)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// EncryptChaCha20Poly1305 performs ChaCha20-Poly1305 encryption.

func (c *CryptoModern) EncryptChaCha20Poly1305(plaintext, key, aad []byte) (*EncryptedData, error) {
	if len(key) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("invalid key size: expected %d, got %d", chacha20poly1305.KeySize, len(key))
	}

	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Generate nonce.

	nonce := make([]byte, chacha20poly1305.NonceSize)

	if _, err := c.entropyPool.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt with AAD.

	ciphertext := aead.Seal(nil, nonce, plaintext, aad)

	return &EncryptedData{
		Algorithm: "ChaCha20-Poly1305",

		Ciphertext: ciphertext,

		Nonce: nonce,

		AAD: aad,

		Created: time.Now(),

		Version: 1,
	}, nil
}

// DecryptChaCha20Poly1305 performs ChaCha20-Poly1305 decryption.

func (c *CryptoModern) DecryptChaCha20Poly1305(data *EncryptedData, key []byte) ([]byte, error) {
	if data.Algorithm != "ChaCha20-Poly1305" {
		return nil, fmt.Errorf("invalid algorithm: expected ChaCha20-Poly1305, got %s", data.Algorithm)
	}

	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	plaintext, err := aead.Open(nil, data.Nonce, data.Ciphertext, data.AAD)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// GenerateEd25519KeyPair generates a new Ed25519 key pair.

func (c *CryptoModern) GenerateEd25519KeyPair() (*Ed25519KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(c.entropyPool)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Ed25519 key pair: %w", err)
	}

	// Generate key ID.

	h := sha256.Sum256(pub)

	keyID := base64.URLEncoding.EncodeToString(h[:8])

	return &Ed25519KeyPair{
		PublicKey: pub,

		PrivateKey: priv,

		Created: time.Now(),

		ID: keyID,
	}, nil
}

// SignEd25519 creates an Ed25519 signature.

func (c *CryptoModern) SignEd25519(message []byte, privateKey ed25519.PrivateKey) ([]byte, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid private key size")
	}

	signature := ed25519.Sign(privateKey, message)

	return signature, nil
}

// VerifyEd25519 verifies an Ed25519 signature.

func (c *CryptoModern) VerifyEd25519(message, signature []byte, publicKey ed25519.PublicKey) bool {
	if len(publicKey) != ed25519.PublicKeySize {
		return false
	}

	return ed25519.Verify(publicKey, message, signature)
}

// DeriveKeyHKDF derives a key using HKDF.

func (c *CryptoModern) DeriveKeyHKDF(secret, salt []byte, length int) ([]byte, error) {
	if length > 255*sha256.Size {
		return nil, errors.New("requested key length too long for HKDF")
	}

	// If no salt provided, use a zero salt.

	if salt == nil {
		salt = make([]byte, sha256.Size)
	}

	// Create HKDF reader.

	hkdfReader := hkdf.New(sha256.New, secret, salt, c.kdfParams.HKDFInfo)

	// Derive key.

	key := make([]byte, length)

	if _, err := io.ReadFull(hkdfReader, key); err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}

	return key, nil
}

// DeriveKeyPBKDF2 derives a key using PBKDF2.

func (c *CryptoModern) DeriveKeyPBKDF2(password, salt []byte, keyLen int) []byte {
	return pbkdf2.Key(password, salt, c.kdfParams.PBKDF2Iterations, keyLen, sha256.New)
}

// DeriveKeyArgon2 derives a key using Argon2id.

func (c *CryptoModern) DeriveKeyArgon2(password, salt []byte) []byte {
	return argon2.IDKey(

		password,

		salt,

		c.kdfParams.Argon2Time,

		c.kdfParams.Argon2Memory,

		c.kdfParams.Argon2Threads,

		c.kdfParams.Argon2KeyLen,
	)
}

// DeriveKeyScrypt derives a key using scrypt.

func (c *CryptoModern) DeriveKeyScrypt(password, salt []byte, keyLen int) ([]byte, error) {
	return scrypt.Key(

		password,

		salt,

		c.kdfParams.ScryptN,

		c.kdfParams.ScryptR,

		c.kdfParams.ScryptP,

		keyLen,
	)
}

// GenerateSecureRandom generates cryptographically secure random bytes.

func (c *CryptoModern) GenerateSecureRandom(length int) ([]byte, error) {
	buf := make([]byte, length)

	if _, err := c.entropyPool.Read(buf); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	return buf, nil
}

// SecureCompare performs constant-time comparison.

func (c *CryptoModern) SecureCompare(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

// SecureClear overwrites sensitive data in memory.

func SecureClear(data []byte) {
	for i := range data {
		data[i] = 0
	}
}

// HashPassword creates a secure password hash using Argon2id.

func (c *CryptoModern) HashPassword(password string) (string, error) {
	// Generate salt.

	salt := make([]byte, 16)

	if _, err := c.entropyPool.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key using Argon2id.

	hash := c.DeriveKeyArgon2([]byte(password), salt)

	// Encode as base64 with parameters.

	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",

		argon2.Version,

		c.kdfParams.Argon2Memory,

		c.kdfParams.Argon2Time,

		c.kdfParams.Argon2Threads,

		base64.RawStdEncoding.EncodeToString(salt),

		base64.RawStdEncoding.EncodeToString(hash),
	)

	return encoded, nil
}

// VerifyPassword verifies a password against a hash.

func (c *CryptoModern) VerifyPassword(password, encoded string) (bool, error) {
	// Parse encoded hash.

	var version int

	var memory, time uint32

	var threads uint8

	var salt, hash string

	_, err := fmt.Sscanf(encoded, "$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",

		&version, &memory, &time, &threads, &salt, &hash)
	if err != nil {
		return false, fmt.Errorf("failed to parse encoded hash: %w", err)
	}

	// Decode salt and hash.

	saltBytes, err := base64.RawStdEncoding.DecodeString(salt)
	if err != nil {
		return false, fmt.Errorf("failed to decode salt: %w", err)
	}

	hashBytes, err := base64.RawStdEncoding.DecodeString(hash)
	if err != nil {
		return false, fmt.Errorf("failed to decode hash: %w", err)
	}

	// Compute hash of provided password.

	computedHash := argon2.IDKey([]byte(password), saltBytes, time, memory, threads, uint32(len(hashBytes)))

	// Constant-time comparison.

	return c.SecureCompare(hashBytes, computedHash), nil
}

// CreateEncryptionContext creates a new encryption context.

func (c *CryptoModern) CreateEncryptionContext(id, algorithm string, keySize int) (*EncryptionContext, error) {
	c.mu.Lock()

	defer c.mu.Unlock()

	// Generate key.

	key := make([]byte, keySize)

	if _, err := c.entropyPool.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	ctx := &EncryptionContext{
		Algorithm: algorithm,

		Key: key,

		Created: time.Now(),
	}

	c.contexts[id] = ctx

	return ctx, nil
}

// GetEncryptionContext retrieves an encryption context.

func (c *CryptoModern) GetEncryptionContext(id string) (*EncryptionContext, bool) {
	c.mu.RLock()

	defer c.mu.RUnlock()

	ctx, ok := c.contexts[id]

	return ctx, ok
}

// DeleteEncryptionContext securely deletes an encryption context.

func (c *CryptoModern) DeleteEncryptionContext(id string) {
	c.mu.Lock()

	defer c.mu.Unlock()

	if ctx, ok := c.contexts[id]; ok {

		// Securely clear the key.

		SecureClear(ctx.Key)

		delete(c.contexts, id)

	}
}

// GenerateMAC generates a message authentication code.

func (c *CryptoModern) GenerateMAC(message, key []byte) []byte {
	h := sha512.New()

	h.Write(key)

	h.Write(message)

	return h.Sum(nil)
}

// VerifyMAC verifies a message authentication code.

func (c *CryptoModern) VerifyMAC(message, mac, key []byte) bool {
	expectedMAC := c.GenerateMAC(message, key)

	return c.SecureCompare(mac, expectedMAC)
}
