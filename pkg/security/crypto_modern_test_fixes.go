package security

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
)

// DeriveKey derives a key using the specified method and size (test-compatible signature)
func (c *CryptoModern) DeriveKey(password, salt []byte, method string, keySize int) ([]byte, error) {
	if len(password) == 0 {
		return nil, fmt.Errorf("password cannot be empty")
	}

	if len(salt) == 0 {
		return nil, fmt.Errorf("salt cannot be empty")
	}

	if keySize <= 0 {
		return nil, fmt.Errorf("invalid key size: %d", keySize)
	}

	switch method {
	case "pbkdf2":
		return c.DeriveKeyPBKDF2(password, salt, keySize), nil
	case "argon2":
		key := c.DeriveKeyArgon2(password, salt)
		if len(key) < keySize {
			// Extend key using HKDF if needed
			extended, err := c.DeriveKeyHKDF(key, salt, keySize)
			if err != nil {
				return nil, err
			}
			return extended, nil
		}
		return key[:keySize], nil
	case "scrypt":
		return c.DeriveKeyScrypt(password, salt, keySize)
	default:
		return nil, fmt.Errorf("unsupported key derivation method: %s", method)
	}
}

// Encrypt with signature expected by tests
func (c *CryptoModern) Encrypt(data []byte, key []byte, algorithm string) ([]byte, error) {
	switch algorithm {
	case "AES-GCM", "aes-gcm":
		encrypted, err := c.EncryptAESGCM(data, key, nil)
		if err != nil {
			return nil, err
		}
		// Combine nonce + ciphertext for the test format
		result := append(encrypted.Nonce, encrypted.Ciphertext...)
		return result, nil
	case "ChaCha20-Poly1305", "chacha20-poly1305":
		encrypted, err := c.EncryptChaCha20Poly1305(data, key, nil)
		if err != nil {
			return nil, err
		}
		// Combine nonce + ciphertext for the test format
		result := append(encrypted.Nonce, encrypted.Ciphertext...)
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported encryption algorithm: %s", algorithm)
	}
}

// Decrypt with signature expected by tests
func (c *CryptoModern) Decrypt(data []byte, key []byte, algorithm string) ([]byte, error) {
	// The data parameter contains raw ciphertext from the Encrypt function
	// We need to recreate the EncryptedData structure to use the actual decryption methods
	switch algorithm {
	case "AES-GCM", "aes-gcm":
		// Parse the ciphertext to extract nonce and ciphertext
		if len(data) < 12 { // AES-GCM nonce is 12 bytes
			return nil, fmt.Errorf("invalid encrypted data length")
		}

		// Create EncryptedData structure with nonce and ciphertext
		encData := &EncryptedData{
			Nonce:      data[:12], // First 12 bytes are nonce
			Ciphertext: data[12:], // Rest is ciphertext
			Algorithm:  "AES-GCM",
		}

		decrypted, err := c.DecryptAESGCM(encData, key)
		if err != nil {
			return nil, err
		}
		// Ensure empty slice instead of nil for empty data
		if decrypted == nil {
			return []byte{}, nil
		}
		return decrypted, nil

	case "ChaCha20-Poly1305", "chacha20-poly1305":
		// Parse the ciphertext to extract nonce and ciphertext
		if len(data) < 12 { // ChaCha20Poly1305 nonce is 12 bytes
			return nil, fmt.Errorf("invalid encrypted data length")
		}

		// Create EncryptedData structure with nonce and ciphertext
		encData := &EncryptedData{
			Nonce:      data[:12], // First 12 bytes are nonce
			Ciphertext: data[12:], // Rest is ciphertext
			Algorithm:  "ChaCha20-Poly1305",
		}

		decrypted, err := c.DecryptChaCha20Poly1305(encData, key)
		if err != nil {
			return nil, err
		}
		// Ensure empty slice instead of nil for empty data
		if decrypted == nil {
			return []byte{}, nil
		}
		return decrypted, nil

	default:
		return nil, fmt.Errorf("unsupported decryption algorithm: %s", algorithm)
	}
}

// Sign provides digital signature functionality
func (c *CryptoModern) Sign(message []byte, privateKey interface{}, algorithm string) ([]byte, error) {
	switch algorithm {
<<<<<<< HEAD
	case "Ed25519":
=======
	case "ed25519":
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		if privKey, ok := privateKey.(ed25519.PrivateKey); ok {
			return c.SignEd25519(message, privKey)
		}
		return nil, fmt.Errorf("invalid private key type for Ed25519")
<<<<<<< HEAD
=======
	case "ecdsa-p256", "ecdsa-p384", "rsa-2048", "rsa-3072", "rsa-pss":
		// Simulate unsupported algorithms with a proper error message
		return nil, fmt.Errorf("unsupported key pair algorithm: %s", algorithm)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	default:
		return nil, fmt.Errorf("unsupported signature algorithm: %s", algorithm)
	}
}

// Verify provides signature verification functionality
func (c *CryptoModern) Verify(message, signature []byte, publicKey interface{}, algorithm string) (bool, error) {
	switch algorithm {
<<<<<<< HEAD
	case "Ed25519":
=======
	case "ed25519":
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		if pubKey, ok := publicKey.(ed25519.PublicKey); ok {
			return c.VerifyEd25519(message, signature, pubKey), nil
		}
		return false, fmt.Errorf("invalid public key type for Ed25519")
<<<<<<< HEAD
=======
	case "ecdsa-p256", "ecdsa-p384", "rsa-2048", "rsa-3072", "rsa-pss":
		// Simulate unsupported algorithms with a proper error message
		return false, fmt.Errorf("unsupported key pair algorithm: %s", algorithm)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	default:
		return false, fmt.Errorf("unsupported signature algorithm: %s", algorithm)
	}
}

// Hash provides generic hashing interface
func (c *CryptoModern) Hash(data []byte, algorithm string) ([]byte, error) {
	switch algorithm {
	case "SHA256", "sha256":
		hash := c.generateSHA256(data)
		return hash[:], nil
	case "SHA512", "sha512":
		hash := c.generateSHA512(data)
		return hash[:], nil
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %s", algorithm)
	}
}

// GenerateKeyPair generates a cryptographic key pair
func (c *CryptoModern) GenerateKeyPair(algorithm string) (interface{}, interface{}, error) {
	switch algorithm {
<<<<<<< HEAD
	case "Ed25519":
=======
	case "ed25519":
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		keyPair, err := c.GenerateEd25519KeyPair()
		if err != nil {
			return nil, nil, err
		}
<<<<<<< HEAD
		return keyPair.PublicKey, keyPair.PrivateKey, nil
=======
		return keyPair.PrivateKey, keyPair.PublicKey, nil
	case "ecdsa-p256", "ecdsa-p384", "rsa-2048", "rsa-3072", "rsa-pss":
		// Simulate unsupported algorithms with a proper error message
		return nil, nil, fmt.Errorf("unsupported key pair algorithm: %s", algorithm)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	default:
		return nil, nil, fmt.Errorf("unsupported key pair algorithm: %s", algorithm)
	}
}

// Helper functions for cryptographic operations
func (c *CryptoModern) generateSHA256(data []byte) [32]byte {
	return sha256.Sum256(data)
}

func (c *CryptoModern) generateSHA512(data []byte) [64]byte {
	return sha512.Sum512(data)
}
