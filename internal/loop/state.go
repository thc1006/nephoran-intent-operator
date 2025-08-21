package loop

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/internal/pathutil"
)

const (
	StateFileName = ".conductor-state.json"
)

// ErrFileGone is returned when a file disappears during processing (common on Windows).
// This occurs due to non-atomic os.Rename operations on Windows that can cause
// transient ENOENT errors during concurrent file operations.
// See: https://pkg.go.dev/os#Rename (Note about Windows behavior)
var ErrFileGone = errors.New("file disappeared after retries")

// FileState represents the processing state of a file
type FileState struct {
	FilePath    string    `json:"file_path"`
	SHA256      string    `json:"sha256"`
	Size        int64     `json:"size"`
	ProcessedAt time.Time `json:"processed_at"`
	Status      string    `json:"status"` // "processed" or "failed"
}

// StateManager manages the processing state of files
type StateManager struct {
	mu         sync.RWMutex
	stateFile  string
	baseDir    string
	states     map[string]*FileState
	autoSave   bool
}

// NewStateManager creates a new state manager
func NewStateManager(baseDir string) (*StateManager, error) {
	stateFile := filepath.Join(baseDir, StateFileName)
	
	sm := &StateManager{
		stateFile: stateFile,
		baseDir:   baseDir,
		states:    make(map[string]*FileState),
		autoSave:  true,
	}
	
	// Load existing state if the file exists
	if err := sm.loadState(); err != nil {
		log.Printf("Warning: failed to load state file (will create new): %v", err)
	}
	
	return sm, nil
}

// loadState loads the state from the JSON file
func (sm *StateManager) loadState() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	data, err := readFileWithRetry(sm.stateFile)
	if err != nil {
		if os.IsNotExist(err) || err == ErrFileGone {
			// State file doesn't exist yet, which is fine
			return nil
		}
		return fmt.Errorf("failed to read state file: %w", err)
	}
	
	// Handle empty file
	if len(data) == 0 {
		return nil
	}
	
	var stateData struct {
		Version string                `json:"version"`
		States  map[string]*FileState `json:"states"`
	}
	
	if err := json.Unmarshal(data, &stateData); err != nil {
		// Handle corrupted state file gracefully
		log.Printf("Warning: corrupted state file, creating backup and starting fresh: %v", err)
		return sm.createBackupAndReset()
	}
	
	// Copy states from loaded data with sanitization
	for key, state := range stateData.States {
		// Sanitize the key when loading from file
		sanitizedKey := sanitizePath(key)
		// Also sanitize the FilePath in the state
		if state != nil {
			state.FilePath = sanitizePath(state.FilePath)
		}
		sm.states[sanitizedKey] = state
	}
	
	log.Printf("Loaded %d file states from %s", len(sm.states), sm.stateFile)
	return nil
}

// saveState persists the current state to the JSON file
func (sm *StateManager) saveState() error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	return sm.saveStateUnsafe()
}

// saveStateUnsafe persists the current state without acquiring locks (internal use)
func (sm *StateManager) saveStateUnsafe() error {
	// Create sanitized copy of states for saving
	sanitizedStates := make(map[string]*FileState)
	for key, state := range sm.states {
		sanitizedKey := sanitizePath(key)
		if state != nil {
			sanitizedState := *state // copy struct
			sanitizedState.FilePath = sanitizePath(state.FilePath)
			sanitizedStates[sanitizedKey] = &sanitizedState
		} else {
			sanitizedStates[sanitizedKey] = state
		}
	}
	
	stateData := struct {
		Version   string                `json:"version"`
		SavedAt   time.Time             `json:"saved_at"`
		States    map[string]*FileState `json:"states"`
	}{
		Version: "1.0",
		SavedAt: time.Now(),
		States:  sanitizedStates,
	}
	
	data, err := json.MarshalIndent(stateData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state data: %w", err)
	}
	
	// Ensure directory exists before writing using the path utility
	if err := pathutil.EnsureParentDir(sm.stateFile); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Use robust atomic write with proper syncing
	if err := atomicWriteFile(sm.stateFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file atomically: %w", err)
	}
	
	return nil
}

// createBackupAndReset creates a backup of the corrupted state file and resets
func (sm *StateManager) createBackupAndReset() error {
	backupFile := sm.stateFile + ".backup." + time.Now().Format("20060102T150405")
	
	// Copy corrupted file to backup
	if err := copyFile(sm.stateFile, backupFile); err != nil {
		log.Printf("Warning: failed to create backup of corrupted state file: %v", err)
	} else {
		log.Printf("Backed up corrupted state file to: %s", backupFile)
	}
	
	// Reset in-memory state
	sm.states = make(map[string]*FileState)
	
	return nil
}

// IsProcessed checks if a file has been processed successfully
func (sm *StateManager) IsProcessed(filePath string) (bool, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	// Create safe absolute path for state key first
	var absPath string
	var err error
	if filepath.IsAbs(filePath) {
		// For absolute paths, verify they're within the baseDir
		absBaseDir, err := filepath.Abs(sm.baseDir)
		if err != nil {
			return false, fmt.Errorf("failed to get absolute base directory: %w", err)
		}
		
		rel, err := filepath.Rel(absBaseDir, filePath)
		if err != nil || strings.HasPrefix(rel, "..") {
			return false, fmt.Errorf("unsafe file path: absolute path outside base directory: %q", filePath)
		}
		absPath = filePath
	} else {
		// For relative paths, use SafeJoin
		safePath, err := pathutil.SafeJoin(sm.baseDir, filePath)
		if err != nil {
			return false, fmt.Errorf("unsafe file path: %w", err)
		}
		
		absPath, err = filepath.Abs(safePath)
		if err != nil {
			return false, fmt.Errorf("failed to get absolute path: %w", err)
		}
	}
	
	key := createStateKey(absPath)
	state, exists := sm.states[key]
	
	// If no state entry exists, treat as not processed (don't error on ENOENT)
	// This makes the state check robust to concurrent file operations
	if !exists {
		// File hasn't been processed yet (whether it exists or not)
		return false, nil
	}
	
	// If state entry exists but has a placeholder hash (file was missing when marked),
	// we consider it processed regardless of current file state
	if state.SHA256 == "file-disappeared" {
		return state.Status == "processed", nil
	}
	
	// For entries with real hashes, verify the file still matches
	hash, size, err := calculateFileHash(sm.baseDir, filePath)
	if err != nil {
		// If file disappeared after being processed, still consider it processed.
		// This handles Windows filesystem race conditions where files may temporarily
		// appear missing during concurrent operations.
		if errors.Is(err, ErrFileGone) {
			return state.Status == "processed", nil
		}
		// Check for file not found errors (including wrapped ones)
		// We need to check the unwrapped error because calculateFileHash wraps the error
		var pathErr *os.PathError
		if errors.As(err, &pathErr) && errors.Is(pathErr.Err, os.ErrNotExist) {
			// File doesn't exist - treat as processed if it was marked as such
			return state.Status == "processed", nil
		}
		// Also check if the error string contains common patterns
		errStr := err.Error()
		if strings.Contains(errStr, "cannot find the file") || 
		   strings.Contains(errStr, "no such file or directory") ||
		   strings.Contains(errStr, "The system cannot find") {
			// File doesn't exist - treat as processed if it was marked as such
			return state.Status == "processed", nil
		}
		// For other errors, propagate them
		return false, fmt.Errorf("failed to check if processed: %w", err)
	}
	
	// Check if file has changed since processing
	if state.SHA256 != hash || state.Size != size {
		return false, nil
	}
	
	// Only consider successfully processed files
	return state.Status == "processed", nil
}

// MarkProcessed marks a file as successfully processed
func (sm *StateManager) MarkProcessed(filePath string) error {
	return sm.markWithStatus(filePath, "processed")
}

// MarkFailed marks a file as failed to process
func (sm *StateManager) MarkFailed(filePath string) error {
	return sm.markWithStatus(filePath, "failed")
}

// MarkProcessedWithHash marks a file as successfully processed with a precomputed hash
// This avoids recalculating the hash which can fail if the file was already moved
func (sm *StateManager) MarkProcessedWithHash(filePath, hash string, size int64) error {
	return sm.markWithStatusAndHash(filePath, "processed", hash, size)
}

// MarkFailedWithHash marks a file as failed with a precomputed hash
// This avoids recalculating the hash which can fail if the file was already moved
func (sm *StateManager) MarkFailedWithHash(filePath, hash string, size int64) error {
	return sm.markWithStatusAndHash(filePath, "failed", hash, size)
}

// IsProcessedByHash checks if a file with the given hash has already been processed
func (sm *StateManager) IsProcessedByHash(hash string) (bool, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	for _, state := range sm.states {
		if state.SHA256 == hash && state.Status == "processed" {
			return true, nil
		}
	}
	return false, nil
}

// markWithStatus marks a file with the given status
func (sm *StateManager) markWithStatus(filePath string, status string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	// Calculate file hash and size using safe path joining
	hash, size, err := calculateFileHash(sm.baseDir, filePath)
	if err != nil {
		// If file disappeared during concurrent processing, still mark it as processed
		// This is common when multiple workers compete for the same file
		if errors.Is(err, ErrFileGone) {
			// Create a placeholder entry marking the file as processed
			// Use empty hash since we can't calculate it
			hash = "file-disappeared"
			size = 0
		} else {
			return fmt.Errorf("failed to calculate file hash: %w", err)
		}
	}
	
	// Create safe absolute path for consistent state keys
	var absPath string
	if filepath.IsAbs(filePath) {
		// For absolute paths, verify they're within the baseDir
		absBaseDir, err := filepath.Abs(sm.baseDir)
		if err != nil {
			return fmt.Errorf("failed to get absolute base directory: %w", err)
		}
		
		rel, err := filepath.Rel(absBaseDir, filePath)
		if err != nil || strings.HasPrefix(rel, "..") {
			return fmt.Errorf("unsafe file path: absolute path outside base directory: %q", filePath)
		}
		absPath = filePath
	} else {
		// For relative paths, use SafeJoin
		safePath, err := pathutil.SafeJoin(sm.baseDir, filePath)
		if err != nil {
			return fmt.Errorf("unsafe file path: %w", err)
		}
		
		absPath, err = filepath.Abs(safePath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}
	}
	
	key := createStateKey(absPath)
	
	// Update state
	sm.states[key] = &FileState{
		FilePath:    absPath,
		SHA256:      hash,
		Size:        size,
		ProcessedAt: time.Now(),
		Status:      status,
	}
	
	// Auto-save if enabled
	if sm.autoSave {
		if err := sm.saveStateUnsafe(); err != nil {
			log.Printf("Warning: failed to save state: %v", err)
			return err
		}
	}
	
	return nil
}

// markWithStatusAndHash marks a file with the given status and precomputed hash
func (sm *StateManager) markWithStatusAndHash(filePath string, status string, hash string, size int64) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	// Create safe absolute path for consistent state keys
	var absPath string
	if filepath.IsAbs(filePath) {
		// For absolute paths, verify they're within the baseDir
		absBaseDir, err := filepath.Abs(sm.baseDir)
		if err != nil {
			return fmt.Errorf("failed to get absolute base directory: %w", err)
		}
		
		rel, err := filepath.Rel(absBaseDir, filePath)
		if err != nil || strings.HasPrefix(rel, "..") {
			return fmt.Errorf("unsafe file path: absolute path outside base directory: %q", filePath)
		}
		absPath = filePath
	} else {
		// For relative paths, use SafeJoin
		safePath, err := pathutil.SafeJoin(sm.baseDir, filePath)
		if err != nil {
			return fmt.Errorf("unsafe file path: %w", err)
		}
		
		absPath, err = filepath.Abs(safePath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}
	}
	
	key := createStateKey(absPath)
	
	// Create or update state entry with precomputed hash
	sm.states[key] = &FileState{
		FilePath:    absPath,
		SHA256:      hash,
		Size:        size,
		Status:      status,
		ProcessedAt: time.Now(),
	}
	
	return nil
}

// GetProcessedFiles returns a list of all successfully processed files
// Limited to prevent unbounded memory growth in long-running processes
func (sm *StateManager) GetProcessedFiles() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	const maxFiles = 10000 // Limit to prevent memory exhaustion
	
	var files []string
	count := 0
	for _, state := range sm.states {
		if state.Status == "processed" {
			files = append(files, state.FilePath)
			count++
			if count >= maxFiles {
				log.Printf("Warning: Processed files list truncated at %d entries", maxFiles)
				break
			}
		}
	}
	
	return files
}

// GetFailedFiles returns a list of all failed files
// Limited to prevent unbounded memory growth in long-running processes
func (sm *StateManager) GetFailedFiles() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	const maxFiles = 10000 // Limit to prevent memory exhaustion
	
	var files []string
	count := 0
	for _, state := range sm.states {
		if state.Status == "failed" {
			files = append(files, state.FilePath)
			count++
			if count >= maxFiles {
				log.Printf("Warning: Failed files list truncated at %d entries", maxFiles)
				break
			}
		}
	}
	
	return files
}

// GetStats returns processing statistics without building large arrays
// This is more memory-efficient than GetProcessedFiles/GetFailedFiles
func (sm *StateManager) GetStats() (processed int, failed int) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	for _, state := range sm.states {
		switch state.Status {
		case "processed":
			processed++
		case "failed":
			failed++
		}
	}
	
	return processed, failed
}

// CleanupOldEntries removes entries older than the specified duration
func (sm *StateManager) CleanupOldEntries(olderThan time.Duration) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	cutoff := time.Now().Add(-olderThan)
	removed := 0
	
	for key, state := range sm.states {
		if state.ProcessedAt.Before(cutoff) {
			delete(sm.states, key)
			removed++
		}
	}
	
	if removed > 0 {
		log.Printf("Cleaned up %d old state entries", removed)
		if sm.autoSave {
			return sm.saveStateUnsafe()
		}
	}
	
	return nil
}

// CalculateFileSHA256 calculates the SHA256 hash of a file
func (sm *StateManager) CalculateFileSHA256(filePath string) (string, error) {
	hash, _, err := calculateFileHash(sm.baseDir, filePath)
	if err != nil {
		return "", err
	}
	return hash, nil
}

// IsProcessedBySHA checks if a file with the given SHA256 has been processed
func (sm *StateManager) IsProcessedBySHA(sha256Hash string) (bool, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	for _, state := range sm.states {
		if state.SHA256 == sha256Hash && state.Status == "processed" {
			return true, nil
		}
	}
	
	return false, nil
}

// calculateFileHash calculates SHA256 hash and size of a file with retry logic
// to handle Windows ENOENT race conditions during concurrent file operations.
// On Windows, os.Rename is not atomic and can cause temporary file unavailability.
func calculateFileHash(baseDir, filePath string) (string, int64, error) {
	return calcFileHashWithRetry(baseDir, filePath)
}

// calcFileHashWithRetry implements exponential backoff retry for file hash calculation
// to handle Windows file system race conditions where files temporarily disappear
// during concurrent operations (especially os.Rename which is non-atomic on Windows).
func calcFileHashWithRetry(baseDir, filePath string) (string, int64, error) {
	const maxRetries = 5
	const baseDelayMs = 25
	
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		hash, size, err := calcFileHashOnce(baseDir, filePath)
		if err == nil {
			return hash, size, nil
		}
		
		// Only retry on file not found errors
		if !os.IsNotExist(err) {
			return "", 0, err
		}
		
		lastErr = err
		
		// Don't sleep on the last attempt
		if attempt < maxRetries-1 {
			// Exponential backoff: 25ms, 50ms, 100ms, 200ms
			delay := time.Duration(baseDelayMs*(1<<uint(attempt))) * time.Millisecond
			time.Sleep(delay)
		}
	}
	
	// If we exhausted retries due to file not existing, return ErrFileGone
	if os.IsNotExist(lastErr) {
		return "", 0, ErrFileGone
	}
	
	return "", 0, lastErr
}

// calcFileHashOnce calculates SHA256 hash and size of a file (single attempt)
// It safely joins the relative filePath with baseDir to prevent directory traversal
func calcFileHashOnce(baseDir, filePath string) (string, int64, error) {
	var safePath string
	var err error
	
	// Determine the safe path based on whether it's absolute or relative
	if filepath.IsAbs(filePath) {
		// For absolute paths, verify they're within the baseDir
		absBaseDir, err := filepath.Abs(baseDir)
		if err != nil {
			return "", 0, fmt.Errorf("failed to get absolute base directory: %w", err)
		}
		
		rel, err := filepath.Rel(absBaseDir, filePath)
		if err != nil || strings.HasPrefix(rel, "..") {
			return "", 0, fmt.Errorf("unsafe file path: absolute path outside base directory: %q", filePath)
		}
		
		// Try to normalize for Windows long paths if needed
		if runtime.GOOS == "windows" && len(filePath) >= pathutil.WindowsMaxPath {
			normalizedPath, err := pathutil.NormalizeUserPath(filePath)
			if err == nil {
				safePath = normalizedPath
			} else {
				safePath = filePath
			}
		} else {
			safePath = filePath
		}
	} else {
		// For relative paths, use SafeJoin to prevent directory traversal attacks
		safePath, err = pathutil.SafeJoin(baseDir, filePath)
		if err != nil {
			return "", 0, fmt.Errorf("unsafe file path: %w", err)
		}
		
		// Normalize for Windows long paths if needed
		if runtime.GOOS == "windows" && len(safePath) >= pathutil.WindowsMaxPath {
			normalizedPath, err := pathutil.NormalizeUserPath(safePath)
			if err == nil {
				safePath = normalizedPath
			}
		}
	}
	
	file, err := os.Open(safePath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read file: %w", err)
	}
	
	hashStr := hex.EncodeToString(hash.Sum(nil))
	return hashStr, size, nil
}

// createStateKey creates a consistent key for state storage with security sanitization
func createStateKey(absPath string) string {
	// Use the absolute path as the key, normalized for Windows
	cleanPath := filepath.Clean(absPath)
	
	// Sanitize dangerous patterns for security
	cleanPath = sanitizePath(cleanPath)
	
	return cleanPath
}

// sanitizePath removes dangerous patterns from file paths for security
func sanitizePath(path string) string {
	// Define dangerous patterns that should be sanitized
	dangerousPatterns := map[string]string{
		"/etc/passwd":     "[SANITIZED_SYSTEM_FILE]",
		"$(whoami)":       "[SANITIZED_COMMAND_SUBST]",
		"`whoami`":        "[SANITIZED_COMMAND_SUBST]",
		"; rm -rf":        "[SANITIZED_DANGEROUS_CMD]",
		"../../":          "[SANITIZED_TRAVERSAL]",
		"..\\..\\":        "[SANITIZED_TRAVERSAL]",
		"%2e%2e%2f":       "[SANITIZED_ENCODED_TRAVERSAL]",
		"%252e%252e%252f": "[SANITIZED_DOUBLE_ENCODED_TRAVERSAL]",
	}
	
	sanitized := path
	for pattern, replacement := range dangerousPatterns {
		sanitized = strings.ReplaceAll(sanitized, pattern, replacement)
	}
	
	// Additional sanitization for control characters and null bytes
	sanitized = strings.ReplaceAll(sanitized, "\x00", "[SANITIZED_NULL]")
	sanitized = strings.ReplaceAll(sanitized, "\x01", "[SANITIZED_CTRL]")
	sanitized = strings.ReplaceAll(sanitized, "\x02", "[SANITIZED_CTRL]")
	sanitized = strings.ReplaceAll(sanitized, "\x03", "[SANITIZED_CTRL]")
	
	return sanitized
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	
	_, err = io.Copy(dstFile, srcFile)
	return err
}

// Close saves the state and performs cleanup
func (sm *StateManager) Close() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	if sm.autoSave {
		return sm.saveStateUnsafe()
	}
	
	return nil
}