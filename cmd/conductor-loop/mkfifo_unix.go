//go:build !windows
// +build !windows

package main

import "syscall"

// mkfifo wraps syscall.Mkfifo for Unix systems
func mkfifo(path string, mode uint32) error {
	return syscall.Mkfifo(path, mode)
}
