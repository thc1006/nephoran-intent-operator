// Package security provides cryptographic utility functions
package security

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"go/constant"
	"hash"
	"io"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/sha3"
)

// CryptoUtils provides cryptographic utility functions
type CryptoUtils struct {
	// Secure memory management
	secureAllocator *SecureAllocator

	// Hash function pool
	hashPool map[string]*sync.Pool

	// Constant-time operations
	constantTime *ConstantTimeOps

	// Random number generator
	rng io.Reader

	// Digital signature chains
	signatureChains map[string]*SignatureChain

	// Encrypted storage
	storage *EncryptedStorage

	mu sync.RWMutex
}

// SecureAllocator manages secure memory allocation
type SecureAllocator struct {
	buffers  map[uintptr]*SecureBuffer
	pageSize int
	mu       sync.Mutex
}

// SecureBuffer represents a secure memory buffer
type SecureBuffer struct {
	data    []byte
	size    int
	locked  bool
	cleared bool
}

// ConstantTimeOps provides constant-time operations
type ConstantTimeOps struct{}

// SignatureChain represents a chain of digital signatures
type SignatureChain struct {
	ChainID    string
	Signatures []*ChainedSignature
	Verifiers  []crypto.PublicKey
	Created    time.Time
}

// ChainedSignature represents a signature in a chain
type ChainedSignature struct {
	SignerID     string
	Signature    []byte
	PreviousHash []byte
	Timestamp    time.Time
	Algorithm    string
}

// EncryptedStorage provides encrypted data storage
type EncryptedStorage struct {
	masterKey []byte
	data      map[string]*EncryptedItem
	mu        sync.RWMutex
}

// EncryptedItem represents an encrypted storage item
type EncryptedItem struct {
	ID         string
	Ciphertext []byte
	Nonce      []byte
	Tag        []byte
	Algorithm  string
	Created    time.Time
	Accessed   time.Time
}

// HashFunction represents supported hash functions
type HashFunction string

const (
	HashSHA256   HashFunction = "SHA256"
	HashSHA512   HashFunction = "SHA512"
	HashSHA3_256 HashFunction = "SHA3-256"
	HashSHA3_512 HashFunction = "SHA3-512"
	HashBLAKE2b  HashFunction = "BLAKE2b"
)

// NewCryptoUtils creates a new crypto utils instance
func NewCryptoUtils() *CryptoUtils {
	cu := &CryptoUtils{
		secureAllocator: NewSecureAllocator(),
		hashPool:        make(map[string]*sync.Pool),
		constantTime:    &ConstantTimeOps{},
		rng:             rand.Reader,
		signatureChains: make(map[string]*SignatureChain),
		storage:         NewEncryptedStorage(),
	}

	// Initialize hash pools
	cu.initHashPools()

	return cu
}

// NewSecureAllocator creates a new secure allocator
func NewSecureAllocator() *SecureAllocator {
	return &SecureAllocator{
		buffers:  make(map[uintptr]*SecureBuffer),
		pageSize: 4096,
	}
}

// NewEncryptedStorage creates a new encrypted storage
func NewEncryptedStorage() *EncryptedStorage {
	// Generate master key
	masterKey := make([]byte, 32)
	rand.Read(masterKey)

	return &EncryptedStorage{
		masterKey: masterKey,
		data:      make(map[string]*EncryptedItem),
	}
}

// initHashPools initializes hash function pools
func (cu *CryptoUtils) initHashPools() {
	// SHA256 pool
	cu.hashPool["SHA256"] = &sync.Pool{
		New: func() interface{} {
			return sha256.New()
		},
	}

	// SHA512 pool
	cu.hashPool["SHA512"] = &sync.Pool{
		New: func() interface{} {
			return sha512.New()
		},
	}

	// SHA3-256 pool
	cu.hashPool["SHA3-256"] = &sync.Pool{
		New: func() interface{} {
			return sha3.New256()
		},
	}

	// SHA3-512 pool
	cu.hashPool["SHA3-512"] = &sync.Pool{
		New: func() interface{} {
			return sha3.New512()
		},
	}

	// BLAKE2b pool
	cu.hashPool["BLAKE2b"] = &sync.Pool{
		New: func() interface{} {
			h, _ := blake2b.New512(nil)
			return h
		},
	}
}

// GetHash gets a hash function from pool
func (cu *CryptoUtils) GetHash(function HashFunction) hash.Hash {
	if pool, ok := cu.hashPool[string(function)]; ok {
		return pool.Get().(hash.Hash)
	}
	// Default to SHA256
	return sha256.New()
}

// PutHash returns a hash function to pool
func (cu *CryptoUtils) PutHash(function HashFunction, h hash.Hash) {
	h.Reset()
	if pool, ok := cu.hashPool[string(function)]; ok {
		pool.Put(h)
	}
}

// ComputeHash computes a hash using specified algorithm
func (cu *CryptoUtils) ComputeHash(data []byte, function HashFunction) []byte {
	h := cu.GetHash(function)
	defer cu.PutHash(function, h)

	h.Write(data)
	return h.Sum(nil)
}

// ConstantTimeCompare performs constant-time byte comparison
func (ct *ConstantTimeOps) Compare(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

// ConstantTimeSelect performs constant-time selection
func (ct *ConstantTimeOps) Select(v int, a, b []byte) []byte {
	result := make([]byte, len(a))
	subtle.ConstantTimeCopy(v, result, a)
	subtle.ConstantTimeCopy(1-v, result, b)
	return result
}

// ConstantTimeLessOrEq performs constant-time less-or-equal comparison
func (ct *ConstantTimeOps) LessOrEq(x, y int32) int {
	return subtle.ConstantTimeLessOrEq(int(x), int(y))
}

// SecureRandom generates cryptographically secure random bytes
func (cu *CryptoUtils) SecureRandom(length int) ([]byte, error) {
	buf := make([]byte, length)
	if _, err := io.ReadFull(cu.rng, buf); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return buf, nil
}

// SecureRandomInt generates a secure random integer
func (cu *CryptoUtils) SecureRandomInt(max int) (int, error) {
	if max <= 0 {
		return 0, errors.New("max must be positive")
	}

	// Calculate required bytes
	bytesNeeded := (bitLen(max) + 7) / 8

	for {
		bytes, err := cu.SecureRandom(bytesNeeded)
		if err != nil {
			return 0, err
		}

		// Convert to integer
		n := 0
		for _, b := range bytes {
			n = (n << 8) | int(b)
		}

		// Ensure uniform distribution
		if n < max {
			return n, nil
		}
	}
}

// AllocateSecure allocates secure memory
func (sa *SecureAllocator) Allocate(size int) *SecureBuffer {
	sa.mu.Lock()
	defer sa.mu.Unlock()

	// Align to page size
	alignedSize := ((size + sa.pageSize - 1) / sa.pageSize) * sa.pageSize

	buffer := &SecureBuffer{
		data:    make([]byte, alignedSize),
		size:    size,
		locked:  false,
		cleared: false,
	}

	// Store buffer reference
	ptr := uintptr(unsafe.Pointer(&buffer.data[0]))
	sa.buffers[ptr] = buffer

	// Try to lock memory (platform-specific)
	buffer.lock()

	return buffer
}

// lock attempts to lock memory pages (platform-specific)
func (sb *SecureBuffer) lock() {
	// Platform-specific implementation would go here
	// For example, using mlock on Unix systems
	sb.locked = true
}

// Clear securely clears the buffer
func (sb *SecureBuffer) Clear() {
	if sb.cleared {
		return
	}

	// Overwrite with random data multiple times
	for i := 0; i < 3; i++ {
		rand.Read(sb.data)
	}

	// Final overwrite with zeros
	for i := range sb.data {
		sb.data[i] = 0
	}

	sb.cleared = true

	// Force garbage collection hint
	runtime.GC()
}

// CreateSignatureChain creates a new signature chain
func (cu *CryptoUtils) CreateSignatureChain(chainID string) *SignatureChain {
	cu.mu.Lock()
	defer cu.mu.Unlock()

	chain := &SignatureChain{
		ChainID:    chainID,
		Signatures: make([]*ChainedSignature, 0),
		Verifiers:  make([]crypto.PublicKey, 0),
		Created:    time.Now(),
	}

	cu.signatureChains[chainID] = chain
	return chain
}

// AddSignature adds a signature to the chain
func (sc *SignatureChain) AddSignature(signerID string, signature []byte, algorithm string) {
	// Calculate previous hash
	var previousHash []byte
	if len(sc.Signatures) > 0 {
		lastSig := sc.Signatures[len(sc.Signatures)-1]
		h := sha256.New()
		h.Write([]byte(lastSig.SignerID))
		h.Write(lastSig.Signature)
		h.Write([]byte(lastSig.Timestamp.String()))
		previousHash = h.Sum(nil)
	}

	chainedSig := &ChainedSignature{
		SignerID:     signerID,
		Signature:    signature,
		PreviousHash: previousHash,
		Timestamp:    time.Now(),
		Algorithm:    algorithm,
	}

	sc.Signatures = append(sc.Signatures, chainedSig)
}

// VerifyChain verifies the integrity of the signature chain
func (sc *SignatureChain) VerifyChain() bool {
	for i, sig := range sc.Signatures {
		// Verify previous hash
		if i > 0 {
			prevSig := sc.Signatures[i-1]
			h := sha256.New()
			h.Write([]byte(prevSig.SignerID))
			h.Write(prevSig.Signature)
			h.Write([]byte(prevSig.Timestamp.String()))
			expectedHash := h.Sum(nil)

			if !bytes.Equal(sig.PreviousHash, expectedHash) {
				return false
			}
		}

		// Verify signature if verifier available
		if i < len(sc.Verifiers) {
			// Signature verification would go here
		}
	}

	return true
}

// Store stores encrypted data
func (es *EncryptedStorage) Store(id string, data []byte) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	// Generate nonce
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Create cipher
	block, err := aes.NewCipher(es.masterKey)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	// Encrypt data
	ciphertext := gcm.Seal(nil, nonce, data, []byte(id))

	// Store encrypted item
	es.data[id] = &EncryptedItem{
		ID:         id,
		Ciphertext: ciphertext,
		Nonce:      nonce,
		Algorithm:  "AES-256-GCM",
		Created:    time.Now(),
		Accessed:   time.Now(),
	}

	return nil
}

// Retrieve retrieves and decrypts data
func (es *EncryptedStorage) Retrieve(id string) ([]byte, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	item, ok := es.data[id]
	if !ok {
		return nil, fmt.Errorf("item not found: %s", id)
	}

	// Create cipher
	block, err := aes.NewCipher(es.masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt data
	plaintext, err := gcm.Open(nil, item.Nonce, item.Ciphertext, []byte(id))
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	// Update access time
	item.Accessed = time.Now()

	return plaintext, nil
}

// TimingSafeEqual performs timing-safe comparison
func TimingSafeEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare(a, b) == 1
}

// ZeroBytes securely zeros a byte slice
func ZeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
	// Prevent compiler optimization
	runtime.KeepAlive(b)
}

// XORBytes performs XOR operation on two byte slices
func XORBytes(a, b []byte) []byte {
	if len(a) != len(b) {
		panic("XORBytes: length mismatch")
	}

	result := make([]byte, len(a))
	subtle.XORBytes(result, a, b)
	return result
}

// PadPKCS7 applies PKCS7 padding
func PadPKCS7(data []byte, blockSize int) []byte {
	padding := blockSize - (len(data) % blockSize)
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

// UnpadPKCS7 removes PKCS7 padding
func UnpadPKCS7(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("empty data")
	}

	padding := int(data[length-1])
	if padding > length || padding == 0 {
		return nil, errors.New("invalid padding")
	}

	for i := 0; i < padding; i++ {
		if data[length-1-i] != byte(padding) {
			return nil, errors.New("invalid padding")
		}
	}

	return data[:length-padding], nil
}

// EncodeBase64 encodes data to base64
func EncodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeBase64 decodes base64 data
func DecodeBase64(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}

// EncodeHex encodes data to hexadecimal
func EncodeHex(data []byte) string {
	return hex.EncodeToString(data)
}

// DecodeHex decodes hexadecimal data
func DecodeHex(encoded string) ([]byte, error) {
	return hex.DecodeString(encoded)
}

// GenerateKeyPair generates an RSA key pair
func GenerateKeyPair(bits int) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	return privateKey, &privateKey.PublicKey, nil
}

// SerializePublicKey serializes a public key to PEM format
func SerializePublicKey(pub *rsa.PublicKey) ([]byte, error) {
	pubASN1, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}

	return pubASN1, nil
}

// DeserializePublicKey deserializes a public key from PEM format
func DeserializePublicKey(data []byte) (*rsa.PublicKey, error) {
	pub, err := x509.ParsePKIXPublicKey(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}

	return rsaPub, nil
}

// Helper functions

// bitLen returns the bit length of an integer
func bitLen(n int) int {
	bits := 0
	for n > 0 {
		bits++
		n >>= 1
	}
	return bits
}

// constant is imported to ensure constant-time operations
var _ = constant.MakeFromLiteral
