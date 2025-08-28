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
	"strings"
	"sync"
	"time"
)

const (
	// StateFileName holds statefilename value.
	StateFileName = ".conductor-state.json"
)

// FileState represents the processing state of a file.
type FileState struct {
	FilePath    string    `json:"file_path"`
	SHA256      string    `json:"sha256"`
	Size        int64     `json:"size"`
	ProcessedAt time.Time `json:"processed_at"`
	Status      string    `json:"status"` // "processed" or "failed"
}

// StateManager manages the processing state of files.
type StateManager struct {
	mu        sync.RWMutex
	stateFile string
	states    map[string]*FileState
	autoSave  bool
}

// NewStateManager creates a new state manager.
func NewStateManager(baseDir string) (*StateManager, error) {
	stateFile := filepath.Join(baseDir, StateFileName)

	sm := &StateManager{
		stateFile: stateFile,
		states:    make(map[string]*FileState),
		autoSave:  true,
	}

	// Load existing state if the file exists.
	if err := sm.loadState(); err != nil {
		log.Printf("Warning: failed to load state file (will create new): %v", err)
	}

	return sm, nil
}

// loadState loads the state from the JSON file.
func (sm *StateManager) loadState() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	data, err := os.ReadFile(sm.stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			// State file doesn't exist yet, which is fine.
			return nil
		}
		return fmt.Errorf("failed to read state file: %w", err)
	}

	// Handle empty file.
	if len(data) == 0 {
		return nil
	}

	var stateData struct {
		Version string                `json:"version"`
		States  map[string]*FileState `json:"states"`
	}

	if err := json.Unmarshal(data, &stateData); err != nil {
		// Handle corrupted state file gracefully.
		log.Printf("Warning: corrupted state file, creating backup and starting fresh: %v", err)
		return sm.createBackupAndReset()
	}

	// Copy states from loaded data with sanitization.
	for key, state := range stateData.States {
		// Sanitize the key when loading from file.
		sanitizedKey := sanitizePath(key)
		// Also sanitize the FilePath in the state.
		if state != nil {
			state.FilePath = sanitizePath(state.FilePath)
		}
		sm.states[sanitizedKey] = state
	}

	log.Printf("Loaded %d file states from %s", len(sm.states), sm.stateFile)
	return nil
}

// saveState persists the current state to the JSON file.
func (sm *StateManager) saveState() error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.saveStateUnsafe()
}

// saveStateUnsafe persists the current state without acquiring locks (internal use).
func (sm *StateManager) saveStateUnsafe() error {
	// Create sanitized copy of states for saving.
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
		Version string                `json:"version"`
		SavedAt time.Time             `json:"saved_at"`
		States  map[string]*FileState `json:"states"`
	}{
		Version: "1.0",
		SavedAt: time.Now(),
		States:  sanitizedStates,
	}

	data, err := json.MarshalIndent(stateData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state data: %w", err)
	}

	// Write atomically by using a temporary file.
	tempFile := sm.stateFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0o640); err != nil {
		return fmt.Errorf("failed to write temporary state file: %w", err)
	}

	// Atomic rename on Windows - use os.Rename which handles cross-device moves.
	if err := os.Rename(tempFile, sm.stateFile); err != nil {
		os.Remove(tempFile) // Clean up temp file on failure
		return fmt.Errorf("failed to rename temporary state file: %w", err)
	}

	return nil
}

// createBackupAndReset creates a backup of the corrupted state file and resets.
func (sm *StateManager) createBackupAndReset() error {
	backupFile := sm.stateFile + ".backup." + time.Now().Format("20060102T150405")

	// Copy corrupted file to backup.
	if err := copyFile(sm.stateFile, backupFile); err != nil {
		log.Printf("Warning: failed to create backup of corrupted state file: %v", err)
	} else {
		log.Printf("Backed up corrupted state file to: %s", backupFile)
	}

	// Reset in-memory state.
	sm.states = make(map[string]*FileState)

	return nil
}

// IsProcessed checks if a file has been processed successfully.
// Returns false gracefully if the file doesn't exist (common in rapid file churn scenarios).
func (sm *StateManager) IsProcessed(filePath string) (bool, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Calculate current file hash and size.
	hash, size, err := calculateFileHash(filePath)
	if err != nil {
		// Handle file gone gracefully - not an error, just return false.
		if errors.Is(err, ErrFileGone) {
			return false, nil
		}
		// Other errors are still genuine errors.
		return false, fmt.Errorf("failed to calculate file hash: %w", err)
	}

	// Create state key from absolute path.
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to get absolute path: %w", err)
	}

	key := createStateKey(absPath)
	state, exists := sm.states[key]

	if !exists {
		return false, nil
	}

	// Check if file has changed since processing.
	if state.SHA256 != hash || state.Size != size {
		return false, nil
	}

	// Only consider successfully processed files.
	return state.Status == "processed", nil
}

// MarkProcessed marks a file as successfully processed.
func (sm *StateManager) MarkProcessed(filePath string) error {
	return sm.markWithStatus(filePath, "processed")
}

// MarkFailed marks a file as failed to process.
func (sm *StateManager) MarkFailed(filePath string) error {
	return sm.markWithStatus(filePath, "failed")
}

// markWithStatus marks a file with the given status.
func (sm *StateManager) markWithStatus(filePath, status string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Calculate file hash and size.
	hash, size, err := calculateFileHash(filePath)
	if err != nil {
		// Handle file gone gracefully - file disappeared during processing.
		if errors.Is(err, ErrFileGone) {
			return ErrFileGone
		}
		return fmt.Errorf("failed to calculate file hash: %w", err)
	}

	// Create absolute path for consistent state keys.
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	key := createStateKey(absPath)

	// Update state.
	sm.states[key] = &FileState{
		FilePath:    absPath,
		SHA256:      hash,
		Size:        size,
		ProcessedAt: time.Now(),
		Status:      status,
	}

	// Auto-save if enabled.
	if sm.autoSave {
		if err := sm.saveStateUnsafe(); err != nil {
			log.Printf("Warning: failed to save state: %v", err)
			return err
		}
	}

	return nil
}

// GetProcessedFiles returns a list of all successfully processed files.
func (sm *StateManager) GetProcessedFiles() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var files []string
	for _, state := range sm.states {
		if state.Status == "processed" {
			files = append(files, state.FilePath)
		}
	}

	return files
}

// GetFailedFiles returns a list of all failed files.
func (sm *StateManager) GetFailedFiles() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var files []string
	for _, state := range sm.states {
		if state.Status == "failed" {
			files = append(files, state.FilePath)
		}
	}

	return files
}

// CleanupOldEntries removes entries older than the specified duration.
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

// CalculateFileSHA256 calculates the SHA256 hash of a file.
func (sm *StateManager) CalculateFileSHA256(filePath string) (string, error) {
	hash, _, err := calculateFileHash(filePath)
	if err != nil {
		// Keep ErrFileGone as-is for consistency.
		return "", err
	}
	return hash, nil
}

// IsProcessedBySHA checks if a file with the given SHA256 has been processed.
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

// calculateFileHash calculates SHA256 hash and size of a file with retry logic.
func calculateFileHash(filePath string) (string, int64, error) {
	return calculateFileHashWithRetry(filePath, 3, 10*time.Millisecond)
}

// calculateFileHashWithRetry calculates SHA256 hash and size with retry logic for transient failures.
func calculateFileHashWithRetry(filePath string, maxRetries int, baseDelay time.Duration) (string, int64, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		file, err := os.Open(filePath)
		if err != nil {
			// Check if file is truly gone (not transient error).
			if os.IsNotExist(err) {
				// For rapid file churn, return ErrFileGone instead of wrapped error.
				return "", 0, ErrFileGone
			}

			lastErr = err

			// Retry on other errors (like sharing violations on Windows).
			if attempt < maxRetries {
				delay := baseDelay * time.Duration(1<<attempt) // exponential backoff
				time.Sleep(delay)
				continue
			}

			return "", 0, fmt.Errorf("failed to open file after %d attempts: %w", maxRetries+1, lastErr)
		}

		hash := sha256.New()
		size, err := io.Copy(hash, file)
		file.Close()

		if err != nil {
			lastErr = err
			// Retry on read errors.
			if attempt < maxRetries {
				delay := baseDelay * time.Duration(1<<attempt)
				time.Sleep(delay)
				continue
			}
			return "", 0, fmt.Errorf("failed to read file after %d attempts: %w", maxRetries+1, lastErr)
		}

		hashStr := hex.EncodeToString(hash.Sum(nil))
		return hashStr, size, nil
	}

	return "", 0, fmt.Errorf("failed to calculate hash after %d attempts: %w", maxRetries+1, lastErr)
}

// createStateKey creates a consistent key for state storage with security sanitization.
func createStateKey(absPath string) string {
	// Use the absolute path as the key, normalized for Windows.
	cleanPath := filepath.Clean(absPath)

	// Sanitize dangerous patterns for security.
	cleanPath = sanitizePath(cleanPath)

	return cleanPath
}

// sanitizePath removes dangerous patterns from file paths for security.
func sanitizePath(path string) string {
	// Define dangerous patterns that should be sanitized.
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

	// Additional sanitization for control characters and null bytes.
	sanitized = strings.ReplaceAll(sanitized, "\x00", "[SANITIZED_NULL]")
	sanitized = strings.ReplaceAll(sanitized, "\x01", "[SANITIZED_CTRL]")
	sanitized = strings.ReplaceAll(sanitized, "\x02", "[SANITIZED_CTRL]")
	sanitized = strings.ReplaceAll(sanitized, "\x03", "[SANITIZED_CTRL]")

	return sanitized
}

// copyFile copies a file from src to dst.
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

// Close saves the state and performs cleanup.
func (sm *StateManager) Close() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.autoSave {
		return sm.saveStateUnsafe()
	}

	return nil
}
