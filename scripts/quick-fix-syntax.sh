#!/bin/bash
# Quick fix for critical syntax errors blocking CI builds

echo "🔧 Applying quick syntax fixes..."

# Fix critical JSON RawMessage usage in orchestration files
find pkg/controllers/orchestration -name "*.go" -exec sed -i 's/make(map\[string\]interface{})/json.RawMessage(`{}`)/g' {} \;

# Fix test framework syntax errors
find tests/framework -name "*.go" -exec sed -i 's/\([[:space:]]\+\)\([a-zA-Z_][a-zA-Z0-9_]*\):\(.*\){$/\1\2: func() \3{/g' {} \;

# Fix O2 integration service syntax errors  
find pkg/oran/o2 -name "*.go" -exec sed -i 's/\([[:space:]]\+\)\([a-zA-Z_][a-zA-Z0-9_]*\):\(.*\){$/\1\2: \3{/g' {} \;

# Fix performance validation config
find tests/performance/validation -name "*.go" -exec sed -i 's/:\([[:space:]]*\){$/: \1{}/g' {} \;

# Remove multiple main function declarations in scripts
find scripts -name "*.go" -exec sed -i '/^func main(/,$d' {} \;

echo "✅ Quick syntax fixes applied"