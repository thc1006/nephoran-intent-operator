//go:build windows
// +build windows

package main

import "errors"

// mkfifo is not supported on Windows.
// This function is required for build compatibility but should not be called on Windows.

func mkfifo(_ string, _ uint32) error {
	return errors.New("mkfifo not supported on Windows")
}
