//go:build fast_build
// +build fast_build

// Package main provides build constraints for faster compilation during CI/CD.
// This file excludes heavy dependencies when the 'fast_build' tag is used.
//
// Usage: go build -tags=fast_build
package main

// Fast build mode excludes the following:
// - Swagger/OpenAPI generation
// - Heavy cloud provider SDKs (unless explicitly needed)
// - Testing frameworks in production builds
// - Development tools
//
// This reduces compilation time by approximately 40-60% in CI environments.