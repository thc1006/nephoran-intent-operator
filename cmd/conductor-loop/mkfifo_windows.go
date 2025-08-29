//go:build windows

// +build windows




package main



import "errors"



// mkfifo is not supported on Windows.

func mkfifo(path string, mode uint32) error {

	return errors.New("mkfifo not supported on Windows")

}

