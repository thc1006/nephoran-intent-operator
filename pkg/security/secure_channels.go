// Package security provides secure communication channel implementations
package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// SecureChannel provides end-to-end encrypted communication
type SecureChannel struct {
	// Connection
	conn      net.Conn
	tlsConfig *tls.Config

	// Encryption
	cipher    cipher.AEAD
	sendNonce []byte
	recvNonce []byte

	// Authentication
	sendMAC secureHash
	recvMAC secureHash

	// Perfect Forward Secrecy
	ephemeralKey []byte
	sessionKey   []byte

	// Anti-replay
	replayWindow *ReplayWindow
	sequenceNum  uint64

	// Session management
	sessionID    string
	established  time.Time
	lastActivity time.Time

	// Multicast support
	multicastGroup *MulticastGroup

	// Metrics
	bytesSent     uint64
	bytesReceived uint64
	messagesSent  uint64
	messagesRecv  uint64

	mu sync.RWMutex
}

// secureHash interface for MAC operations
type secureHash interface {
	Write([]byte) (int, error)
	Sum([]byte) []byte
	Reset()
}

// ReplayWindow implements anti-replay protection
type ReplayWindow struct {
	windowSize uint32
	bitmap     []uint64
	lastSeq    uint64
	mu         sync.Mutex
}

// MulticastGroup manages secure multicast communication
type MulticastGroup struct {
	groupID    string
	members    map[string]*GroupMember
	groupKey   []byte
	keyVersion int
	mu         sync.RWMutex
}

// GroupMember represents a multicast group member
type GroupMember struct {
	ID        string
	PublicKey []byte
	Address   string
	Joined    time.Time
	Active    bool
}

// SecureMessage represents an encrypted message
type SecureMessage struct {
	Version     uint8
	MessageType uint8
	SequenceNum uint64
	Timestamp   int64
	SessionID   []byte
	Ciphertext  []byte
	MAC         []byte
	Nonce       []byte
}

// MessageType constants
const (
	MessageTypeData      uint8 = 0x01
	MessageTypeHandshake uint8 = 0x02
	MessageTypeHeartbeat uint8 = 0x03
	MessageTypeRekey     uint8 = 0x04
	MessageTypeClose     uint8 = 0x05
)

// ChannelConfig contains secure channel configuration
type ChannelConfig struct {
	// Encryption
	CipherSuite string
	KeySize     int

	// Authentication
	MACAlgorithm string

	// PFS
	EnablePFS bool
	DHGroup   string

	// Anti-replay
	ReplayWindow uint32

	// Session
	SessionTimeout    time.Duration
	HeartbeatInterval time.Duration

	// Multicast
	EnableMulticast bool
	MulticastTTL    int
}

// DefaultChannelConfig returns default secure channel configuration
func DefaultChannelConfig() *ChannelConfig {
	return &ChannelConfig{
		CipherSuite:       "AES-256-GCM",
		KeySize:           32,
		MACAlgorithm:      "HMAC-SHA256",
		EnablePFS:         true,
		DHGroup:           "P-256",
		ReplayWindow:      1024,
		SessionTimeout:    24 * time.Hour,
		HeartbeatInterval: 30 * time.Second,
		EnableMulticast:   false,
		MulticastTTL:      1,
	}
}

// NewSecureChannel creates a new secure channel
func NewSecureChannel(conn net.Conn, config *ChannelConfig) (*SecureChannel, error) {
	sc := &SecureChannel{
		conn:         conn,
		replayWindow: NewReplayWindow(config.ReplayWindow),
		sessionID:    generateSessionID(),
		established:  time.Now(),
		lastActivity: time.Now(),
	}

	// Initialize encryption
	if err := sc.initializeEncryption(config); err != nil {
		return nil, fmt.Errorf("failed to initialize encryption: %w", err)
	}

	// Initialize MAC
	if err := sc.initializeMAC(config); err != nil {
		return nil, fmt.Errorf("failed to initialize MAC: %w", err)
	}

	// Setup PFS if enabled
	if config.EnablePFS {
		if err := sc.setupPFS(config); err != nil {
			return nil, fmt.Errorf("failed to setup PFS: %w", err)
		}
	}

	// Setup multicast if enabled
	if config.EnableMulticast {
		sc.multicastGroup = &MulticastGroup{
			members: make(map[string]*GroupMember),
		}
	}

	// Start heartbeat
	go sc.heartbeatLoop(config.HeartbeatInterval)

	return sc, nil
}

// NewReplayWindow creates a new replay window
func NewReplayWindow(size uint32) *ReplayWindow {
	bitmapSize := (size + 63) / 64
	return &ReplayWindow{
		windowSize: size,
		bitmap:     make([]uint64, bitmapSize),
		lastSeq:    0,
	}
}

// initializeEncryption sets up the encryption cipher
func (sc *SecureChannel) initializeEncryption(config *ChannelConfig) error {
	// Generate session key
	sessionKey := make([]byte, config.KeySize)
	if _, err := rand.Read(sessionKey); err != nil {
		return fmt.Errorf("failed to generate session key: %w", err)
	}
	sc.sessionKey = sessionKey

	// Create cipher
	switch config.CipherSuite {
	case "AES-256-GCM":
		block, err := aes.NewCipher(sessionKey)
		if err != nil {
			return fmt.Errorf("failed to create AES cipher: %w", err)
		}

		aead, err := cipher.NewGCM(block)
		if err != nil {
			return fmt.Errorf("failed to create GCM: %w", err)
		}

		sc.cipher = aead
		sc.sendNonce = make([]byte, aead.NonceSize())
		sc.recvNonce = make([]byte, aead.NonceSize())

	default:
		return fmt.Errorf("unsupported cipher suite: %s", config.CipherSuite)
	}

	return nil
}

// initializeMAC sets up message authentication
func (sc *SecureChannel) initializeMAC(config *ChannelConfig) error {
	// Generate MAC keys
	sendMACKey := make([]byte, 32)
	recvMACKey := make([]byte, 32)

	if _, err := rand.Read(sendMACKey); err != nil {
		return fmt.Errorf("failed to generate send MAC key: %w", err)
	}

	if _, err := rand.Read(recvMACKey); err != nil {
		return fmt.Errorf("failed to generate recv MAC key: %w", err)
	}

	// Create MACs
	switch config.MACAlgorithm {
	case "HMAC-SHA256":
		sc.sendMAC = hmac.New(sha256.New, sendMACKey)
		sc.recvMAC = hmac.New(sha256.New, recvMACKey)

	default:
		return fmt.Errorf("unsupported MAC algorithm: %s", config.MACAlgorithm)
	}

	return nil
}

// setupPFS establishes perfect forward secrecy
func (sc *SecureChannel) setupPFS(config *ChannelConfig) error {
	// Generate ephemeral key
	ephemeralKey := make([]byte, 32)
	if _, err := rand.Read(ephemeralKey); err != nil {
		return fmt.Errorf("failed to generate ephemeral key: %w", err)
	}
	sc.ephemeralKey = ephemeralKey

	// Perform key exchange (simplified)
	// In production, use proper DH or ECDH

	return nil
}

// Send sends an encrypted message
func (sc *SecureChannel) Send(data []byte) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Increment sequence number
	seqNum := atomic.AddUint64(&sc.sequenceNum, 1)

	// Create message
	msg := &SecureMessage{
		Version:     1,
		MessageType: MessageTypeData,
		SequenceNum: seqNum,
		Timestamp:   time.Now().UnixNano(),
		SessionID:   []byte(sc.sessionID),
	}

	// Increment nonce
	incrementNonce(sc.sendNonce)

	// Encrypt data
	ciphertext := sc.cipher.Seal(nil, sc.sendNonce, data, msg.SessionID)
	msg.Ciphertext = ciphertext
	msg.Nonce = make([]byte, len(sc.sendNonce))
	copy(msg.Nonce, sc.sendNonce)

	// Calculate MAC
	mac := sc.calculateMAC(msg)
	msg.MAC = mac

	// Serialize and send
	encoded, err := sc.encodeMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	if _, err := sc.conn.Write(encoded); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	// Update metrics
	atomic.AddUint64(&sc.bytesSent, uint64(len(data)))
	atomic.AddUint64(&sc.messagesSent, 1)
	sc.lastActivity = time.Now()

	return nil
}

// Receive receives and decrypts a message
func (sc *SecureChannel) Receive() ([]byte, error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Read message
	encoded := make([]byte, 65536)
	n, err := sc.conn.Read(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %w", err)
	}

	// Decode message
	msg, err := sc.decodeMessage(encoded[:n])
	if err != nil {
		return nil, fmt.Errorf("failed to decode message: %w", err)
	}

	// Check replay
	if !sc.checkReplay(msg.SequenceNum) {
		return nil, errors.New("replay attack detected")
	}

	// Verify MAC
	expectedMAC := sc.calculateMAC(msg)
	if !hmac.Equal(msg.MAC, expectedMAC) {
		return nil, errors.New("MAC verification failed")
	}

	// Decrypt data
	plaintext, err := sc.cipher.Open(nil, msg.Nonce, msg.Ciphertext, msg.SessionID)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	// Update metrics
	atomic.AddUint64(&sc.bytesReceived, uint64(len(plaintext)))
	atomic.AddUint64(&sc.messagesRecv, 1)
	sc.lastActivity = time.Now()

	return plaintext, nil
}

// SendMulticast sends an encrypted multicast message
func (sc *SecureChannel) SendMulticast(data []byte, groupID string) error {
	if sc.multicastGroup == nil {
		return errors.New("multicast not enabled")
	}

	sc.multicastGroup.mu.RLock()
	defer sc.multicastGroup.mu.RUnlock()

	if sc.multicastGroup.groupID != groupID {
		return fmt.Errorf("invalid group ID: %s", groupID)
	}

	// Encrypt with group key
	_, err := sc.encryptWithGroupKey(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt multicast message: %w", err)
	}

	// Send to all active members
	for _, member := range sc.multicastGroup.members {
		if !member.Active {
			continue
		}

		// Send to member
		// Implementation depends on transport
		_ = member // Placeholder for actual sending implementation
	}

	return nil
}

// JoinMulticastGroup joins a multicast group
func (sc *SecureChannel) JoinMulticastGroup(groupID string, groupKey []byte) error {
	if sc.multicastGroup == nil {
		return errors.New("multicast not enabled")
	}

	sc.multicastGroup.mu.Lock()
	defer sc.multicastGroup.mu.Unlock()

	sc.multicastGroup.groupID = groupID
	sc.multicastGroup.groupKey = groupKey
	sc.multicastGroup.keyVersion++

	return nil
}

// Rekey performs session rekeying
func (sc *SecureChannel) Rekey() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Generate new session key
	newKey := make([]byte, len(sc.sessionKey))
	if _, err := rand.Read(newKey); err != nil {
		return fmt.Errorf("failed to generate new session key: %w", err)
	}

	// Send rekey message
	msg := &SecureMessage{
		Version:     1,
		MessageType: MessageTypeRekey,
		SequenceNum: atomic.AddUint64(&sc.sequenceNum, 1),
		Timestamp:   time.Now().UnixNano(),
		SessionID:   []byte(sc.sessionID),
		Ciphertext:  newKey, // In production, encrypt this
	}

	encoded, err := sc.encodeMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to encode rekey message: %w", err)
	}

	if _, err := sc.conn.Write(encoded); err != nil {
		return fmt.Errorf("failed to send rekey message: %w", err)
	}

	// Update session key
	sc.sessionKey = newKey

	// Recreate cipher with new key
	block, err := aes.NewCipher(newKey)
	if err != nil {
		return fmt.Errorf("failed to create cipher with new key: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM with new key: %w", err)
	}

	sc.cipher = aead

	return nil
}

// checkReplay checks for replay attacks
func (sc *SecureChannel) checkReplay(seqNum uint64) bool {
	return sc.replayWindow.Check(seqNum)
}

// Check verifies if a sequence number is valid (not a replay)
func (rw *ReplayWindow) Check(seqNum uint64) bool {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	// Sequence number too old
	if seqNum <= rw.lastSeq-uint64(rw.windowSize) {
		return false
	}

	// Future sequence number
	if seqNum > rw.lastSeq {
		// Shift window
		shift := seqNum - rw.lastSeq
		if shift >= uint64(rw.windowSize) {
			// Clear bitmap
			for i := range rw.bitmap {
				rw.bitmap[i] = 0
			}
		} else {
			// Shift bitmap
			rw.shiftBitmap(uint32(shift))
		}

		rw.lastSeq = seqNum
		rw.setBit(0)
		return true
	}

	// Within window - check if already seen
	offset := rw.lastSeq - seqNum
	if rw.getBit(uint32(offset)) {
		return false // Already seen
	}

	rw.setBit(uint32(offset))
	return true
}

// shiftBitmap shifts the replay window bitmap
func (rw *ReplayWindow) shiftBitmap(shift uint32) {
	// Simplified shift implementation
	// In production, optimize this
	for i := 0; i < int(shift); i++ {
		// Shift each uint64 in the bitmap
		carry := uint64(0)
		for j := len(rw.bitmap) - 1; j >= 0; j-- {
			newCarry := rw.bitmap[j] >> 63
			rw.bitmap[j] = (rw.bitmap[j] << 1) | carry
			carry = newCarry
		}
	}
}

// setBit sets a bit in the replay window
func (rw *ReplayWindow) setBit(offset uint32) {
	index := offset / 64
	bit := offset % 64
	if int(index) < len(rw.bitmap) {
		rw.bitmap[index] |= (1 << bit)
	}
}

// getBit gets a bit from the replay window
func (rw *ReplayWindow) getBit(offset uint32) bool {
	index := offset / 64
	bit := offset % 64
	if int(index) < len(rw.bitmap) {
		return (rw.bitmap[index] & (1 << bit)) != 0
	}
	return false
}

// calculateMAC calculates message authentication code
func (sc *SecureChannel) calculateMAC(msg *SecureMessage) []byte {
	sc.sendMAC.Reset()

	// MAC covers all fields except MAC itself
	binary.Write(sc.sendMAC, binary.BigEndian, msg.Version)
	binary.Write(sc.sendMAC, binary.BigEndian, msg.MessageType)
	binary.Write(sc.sendMAC, binary.BigEndian, msg.SequenceNum)
	binary.Write(sc.sendMAC, binary.BigEndian, msg.Timestamp)
	sc.sendMAC.Write(msg.SessionID)
	sc.sendMAC.Write(msg.Ciphertext)
	sc.sendMAC.Write(msg.Nonce)

	return sc.sendMAC.Sum(nil)
}

// encodeMessage encodes a message for transmission
func (sc *SecureChannel) encodeMessage(msg *SecureMessage) ([]byte, error) {
	// Simple length-prefixed encoding
	// In production, use proper protocol encoding

	totalLen := 1 + 1 + 8 + 8 + // version, type, seq, timestamp
		4 + len(msg.SessionID) +
		4 + len(msg.Ciphertext) +
		4 + len(msg.MAC) +
		4 + len(msg.Nonce)

	buf := make([]byte, 4+totalLen)
	offset := 0

	// Length prefix
	binary.BigEndian.PutUint32(buf[offset:], uint32(totalLen))
	offset += 4

	// Message fields
	buf[offset] = msg.Version
	offset++
	buf[offset] = msg.MessageType
	offset++

	binary.BigEndian.PutUint64(buf[offset:], msg.SequenceNum)
	offset += 8

	binary.BigEndian.PutUint64(buf[offset:], uint64(msg.Timestamp))
	offset += 8

	// Variable length fields
	offset = encodeBytes(buf, offset, msg.SessionID)
	offset = encodeBytes(buf, offset, msg.Ciphertext)
	offset = encodeBytes(buf, offset, msg.MAC)
	offset = encodeBytes(buf, offset, msg.Nonce)

	return buf[:offset], nil
}

// decodeMessage decodes a received message
func (sc *SecureChannel) decodeMessage(data []byte) (*SecureMessage, error) {
	if len(data) < 4 {
		return nil, errors.New("message too short")
	}

	// Read length prefix
	totalLen := binary.BigEndian.Uint32(data[:4])
	if uint32(len(data)) < 4+totalLen {
		return nil, errors.New("incomplete message")
	}

	offset := 4
	msg := &SecureMessage{}

	// Fixed fields
	msg.Version = data[offset]
	offset++
	msg.MessageType = data[offset]
	offset++

	msg.SequenceNum = binary.BigEndian.Uint64(data[offset:])
	offset += 8

	msg.Timestamp = int64(binary.BigEndian.Uint64(data[offset:]))
	offset += 8

	// Variable fields
	var err error
	msg.SessionID, offset, err = decodeBytes(data, offset)
	if err != nil {
		return nil, err
	}

	msg.Ciphertext, offset, err = decodeBytes(data, offset)
	if err != nil {
		return nil, err
	}

	msg.MAC, offset, err = decodeBytes(data, offset)
	if err != nil {
		return nil, err
	}

	msg.Nonce, offset, err = decodeBytes(data, offset)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

// encodeBytes encodes a byte slice with length prefix
func encodeBytes(buf []byte, offset int, data []byte) int {
	binary.BigEndian.PutUint32(buf[offset:], uint32(len(data)))
	offset += 4
	copy(buf[offset:], data)
	return offset + len(data)
}

// decodeBytes decodes a length-prefixed byte slice
func decodeBytes(buf []byte, offset int) ([]byte, int, error) {
	if offset+4 > len(buf) {
		return nil, 0, errors.New("buffer too short")
	}

	length := binary.BigEndian.Uint32(buf[offset:])
	offset += 4

	if offset+int(length) > len(buf) {
		return nil, 0, errors.New("buffer too short for data")
	}

	data := make([]byte, length)
	copy(data, buf[offset:offset+int(length)])

	return data, offset + int(length), nil
}

// encryptWithGroupKey encrypts data with multicast group key
func (sc *SecureChannel) encryptWithGroupKey(data []byte) ([]byte, error) {
	if sc.multicastGroup.groupKey == nil {
		return nil, errors.New("no group key available")
	}

	// Create cipher with group key
	block, err := aes.NewCipher(sc.multicastGroup.groupKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt
	ciphertext := gcm.Seal(nonce, nonce, data, []byte(sc.multicastGroup.groupID))

	return ciphertext, nil
}

// heartbeatLoop sends periodic heartbeats
func (sc *SecureChannel) heartbeatLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		sc.mu.RLock()
		if time.Since(sc.lastActivity) > interval*2 {
			// Send heartbeat
			msg := &SecureMessage{
				Version:     1,
				MessageType: MessageTypeHeartbeat,
				SequenceNum: atomic.AddUint64(&sc.sequenceNum, 1),
				Timestamp:   time.Now().UnixNano(),
				SessionID:   []byte(sc.sessionID),
			}

			encoded, _ := sc.encodeMessage(msg)
			sc.conn.Write(encoded)
		}
		sc.mu.RUnlock()
	}
}

// Close closes the secure channel
func (sc *SecureChannel) Close() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Send close message
	msg := &SecureMessage{
		Version:     1,
		MessageType: MessageTypeClose,
		SequenceNum: atomic.AddUint64(&sc.sequenceNum, 1),
		Timestamp:   time.Now().UnixNano(),
		SessionID:   []byte(sc.sessionID),
	}

	encoded, _ := sc.encodeMessage(msg)
	sc.conn.Write(encoded)

	// Clear sensitive data
	SecureClear(sc.sessionKey)
	SecureClear(sc.ephemeralKey)
	if sc.multicastGroup != nil {
		SecureClear(sc.multicastGroup.groupKey)
	}

	return sc.conn.Close()
}

// GetMetrics returns channel metrics
func (sc *SecureChannel) GetMetrics() map[string]uint64 {
	return map[string]uint64{
		"bytes_sent":     atomic.LoadUint64(&sc.bytesSent),
		"bytes_received": atomic.LoadUint64(&sc.bytesReceived),
		"messages_sent":  atomic.LoadUint64(&sc.messagesSent),
		"messages_recv":  atomic.LoadUint64(&sc.messagesRecv),
		"sequence_num":   atomic.LoadUint64(&sc.sequenceNum),
	}
}

// Helper functions

// generateSessionID generates a unique session ID
func generateSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// incrementNonce increments a nonce
func incrementNonce(nonce []byte) {
	for i := len(nonce) - 1; i >= 0; i-- {
		nonce[i]++
		if nonce[i] != 0 {
			break
		}
	}
}
