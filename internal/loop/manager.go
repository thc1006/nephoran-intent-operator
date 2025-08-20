package loop

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FileManager manages the organization and movement of processed files
type FileManager struct {
	baseDir      string
	processedDir string
	failedDir    string
	mu           sync.Mutex
}

// NewFileManager creates a new file manager
func NewFileManager(baseDir string) (*FileManager, error) {
	processedDir := filepath.Join(baseDir, "processed")
	failedDir := filepath.Join(baseDir, "failed")
	
	fm := &FileManager{
		baseDir:      baseDir,
		processedDir: processedDir,
		failedDir:    failedDir,
	}
	
	// Create required directories
	if err := fm.ensureDirectories(); err != nil {
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}
	
	return fm, nil
}

// ensureDirectories creates the processed and failed directories if they don't exist
func (fm *FileManager) ensureDirectories() error {
	dirs := []string{fm.processedDir, fm.failedDir}
	
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	
	return nil
}

// MoveToProcessed moves a file to the processed directory
func (fm *FileManager) MoveToProcessed(filePath string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	
	return fm.moveFileToDirectory(filePath, fm.processedDir)
}

// MoveToFailed moves a file to the failed directory and creates an error log
func (fm *FileManager) MoveToFailed(filePath string, errorMsg string) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	
	// Move the file to failed directory
	if err := fm.moveFileToDirectory(filePath, fm.failedDir); err != nil {
		return err
	}
	
	// Create error log file
	return fm.createErrorLog(filePath, errorMsg)
}

// moveFileToDirectory atomically moves a file to a target directory
func (fm *FileManager) moveFileToDirectory(filePath, targetDir string) error {
	// Get the base filename
	fileName := filepath.Base(filePath)
	targetPath := filepath.Join(targetDir, fileName)
	
	// Handle filename conflicts by adding timestamp
	if _, err := os.Stat(targetPath); err == nil {
		timestamp := time.Now().Format("20060102T150405")
		ext := filepath.Ext(fileName)
		baseName := fileName[:len(fileName)-len(ext)]
		fileName = fmt.Sprintf("%s_%s%s", baseName, timestamp, ext)
		targetPath = filepath.Join(targetDir, fileName)
	}
	
	// Perform atomic move - on Windows, we need to handle this carefully
	if err := fm.atomicMove(filePath, targetPath); err != nil {
		return fmt.Errorf("failed to move file %s to %s: %w", filePath, targetPath, err)
	}
	
	log.Printf("Moved file %s to %s", filePath, targetPath)
	return nil
}

// atomicMove performs an atomic file move operation
func (fm *FileManager) atomicMove(src, dst string) error {
	// On Windows, os.Rename is not truly atomic across different volumes,
	// but it's the best we can do within the same volume.
	// For cross-volume moves, we would need copy+delete, but that's not atomic.
	
	// First, try direct rename (works if on same volume)
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	
	// If rename fails, fall back to copy+delete (not atomic but safer than partial state)
	log.Printf("Direct rename failed, falling back to copy+delete for %s -> %s", src, dst)
	return fm.copyAndDelete(src, dst)
}

// copyAndDelete copies a file and then deletes the original (fallback for cross-volume moves)
func (fm *FileManager) copyAndDelete(src, dst string) error {
	// Copy the file
	if err := copyFile(src, dst); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}
	
	// Delete the original
	if err := os.Remove(src); err != nil {
		// If delete fails, try to remove the copy to avoid duplicates
		os.Remove(dst)
		return fmt.Errorf("failed to delete original file: %w", err)
	}
	
	return nil
}

// createErrorLog creates an error log file for a failed processing attempt
func (fm *FileManager) createErrorLog(originalPath, errorMsg string) error {
	fileName := filepath.Base(originalPath)
	logFileName := fileName + ".error.log"
	logFilePath := filepath.Join(fm.failedDir, logFileName)
	
	logEntry := fmt.Sprintf("File: %s\nFailed at: %s\nError: %s\n\n",
		originalPath, time.Now().Format(time.RFC3339), errorMsg)
	
	// Append to existing log file or create new one
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to create error log file: %w", err)
	}
	defer file.Close()
	
	if _, err := file.WriteString(logEntry); err != nil {
		return fmt.Errorf("failed to write to error log file: %w", err)
	}
	
	log.Printf("Created error log: %s", logFilePath)
	return nil
}

// CleanupOldFiles removes files from processed and failed directories that are older than the specified duration
func (fm *FileManager) CleanupOldFiles(olderThan time.Duration) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	
	cutoff := time.Now().Add(-olderThan)
	
	// Cleanup processed files
	processedRemoved, err := fm.cleanupDirectory(fm.processedDir, cutoff)
	if err != nil {
		log.Printf("Warning: failed to cleanup processed directory: %v", err)
	}
	
	// Cleanup failed files
	failedRemoved, err := fm.cleanupDirectory(fm.failedDir, cutoff)
	if err != nil {
		log.Printf("Warning: failed to cleanup failed directory: %v", err)
	}
	
	totalRemoved := processedRemoved + failedRemoved
	if totalRemoved > 0 {
		log.Printf("Cleaned up %d old files (processed: %d, failed: %d)", 
			totalRemoved, processedRemoved, failedRemoved)
	}
	
	return nil
}

// cleanupDirectory removes files older than cutoff time from the specified directory
func (fm *FileManager) cleanupDirectory(dir string, cutoff time.Time) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}
	
	removed := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		filePath := filepath.Join(dir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			log.Printf("Warning: failed to get file info for %s: %v", filePath, err)
			continue
		}
		
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(filePath); err != nil {
				log.Printf("Warning: failed to remove old file %s: %v", filePath, err)
			} else {
				removed++
			}
		}
	}
	
	return removed, nil
}

// GetProcessedFiles returns a list of files in the processed directory
func (fm *FileManager) GetProcessedFiles() ([]string, error) {
	return fm.getFilesInDirectory(fm.processedDir)
}

// GetFailedFiles returns a list of files in the failed directory (excluding .error.log files)
func (fm *FileManager) GetFailedFiles() ([]string, error) {
	files, err := fm.getFilesInDirectory(fm.failedDir)
	if err != nil {
		return nil, err
	}
	
	// Filter out .error.log files
	var failedFiles []string
	for _, file := range files {
		if !strings.HasSuffix(file, ".error.log") {
			failedFiles = append(failedFiles, file)
		}
	}
	return failedFiles, nil
}

// getFilesInDirectory returns a list of regular files in the specified directory
func (fm *FileManager) getFilesInDirectory(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}
	
	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}
	
	return files, nil
}

// GetStats returns statistics about processed and failed files
func (fm *FileManager) GetStats() (ProcessingStats, error) {
	processedFiles, err := fm.GetProcessedFiles()
	if err != nil {
		return ProcessingStats{}, fmt.Errorf("failed to get processed files: %w", err)
	}
	
	failedFiles, err := fm.GetFailedFiles()
	if err != nil {
		return ProcessingStats{}, fmt.Errorf("failed to get failed files: %w", err)
	}
	
	// Categorize failed files into shutdown failures vs real failures
	var shutdownFailedFiles []string
	var realFailedFiles []string
	
	for _, failedFile := range failedFiles {
		// Check error log to determine failure type
		if fm.isShutdownFailure(failedFile) {
			shutdownFailedFiles = append(shutdownFailedFiles, failedFile)
		} else {
			realFailedFiles = append(realFailedFiles, failedFile)
		}
	}
	
	return ProcessingStats{
		ProcessedCount:       len(processedFiles),
		FailedCount:          len(failedFiles),
		ShutdownFailedCount:  len(shutdownFailedFiles),
		RealFailedCount:      len(realFailedFiles),
		ProcessedFiles:       processedFiles,
		FailedFiles:          failedFiles,
		ShutdownFailedFiles:  shutdownFailedFiles,
		RealFailedFiles:      realFailedFiles,
	}, nil
}

// ProcessingStats holds statistics about file processing
type ProcessingStats struct {
	ProcessedCount       int      `json:"processed_count"`
	FailedCount          int      `json:"failed_count"`
	ShutdownFailedCount  int      `json:"shutdown_failed_count"`
	RealFailedCount      int      `json:"real_failed_count"`
	ProcessedFiles       []string `json:"processed_files,omitempty"`
	FailedFiles          []string `json:"failed_files,omitempty"`
	ShutdownFailedFiles  []string `json:"shutdown_failed_files,omitempty"`
	RealFailedFiles      []string `json:"real_failed_files,omitempty"`
}

// IsEmpty checks if both processed and failed directories are empty
func (fm *FileManager) IsEmpty() (bool, error) {
	stats, err := fm.GetStats()
	if err != nil {
		return false, err
	}
	
	return stats.ProcessedCount == 0 && stats.FailedCount == 0, nil
}

// isShutdownFailure checks if a failed file was caused by graceful shutdown
func (fm *FileManager) isShutdownFailure(failedFilePath string) bool {
	// Read the error log file for this failed file
	baseName := filepath.Base(failedFilePath)
	errorLogPath := filepath.Join(fm.baseDir, "failed", baseName+".error.log")  // Fixed: use .error.log extension
	
	errorContent, err := os.ReadFile(errorLogPath)
	if err != nil {
		// If we can't read the error log, assume it's a real failure
		return false
	}
	
	errorMsg := string(errorContent)
	
	// Check for shutdown failure marker or patterns
	return strings.Contains(errorMsg, "SHUTDOWN_FAILURE:") ||
		   strings.Contains(strings.ToLower(errorMsg), "context canceled") ||
		   strings.Contains(strings.ToLower(errorMsg), "context cancelled") ||
		   strings.Contains(strings.ToLower(errorMsg), "signal: killed") ||
		   strings.Contains(strings.ToLower(errorMsg), "signal: interrupt") ||
		   strings.Contains(strings.ToLower(errorMsg), "signal: terminated")
}