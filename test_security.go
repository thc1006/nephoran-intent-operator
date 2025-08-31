package main

import (
    "context"
    "log/slog"
    _ "github.com/thc1006/nephoran-intent-operator/pkg/security"
)

func main() {
    _ = context.Background()
    _ = slog.Default()
}
