#\!/bin/bash
set -e

# Cleanup script for O1 package type consolidation
echo "🔍 Scanning O1 package for type duplicates..."

# List of files to process
declare -a FILES=(
    "pkg/oran/o1/adapter.go"
    "pkg/oran/o1/fault_manager.go"
    "pkg/oran/o1/performance_manager.go"
    "pkg/oran/o1/security/security_types.go"
)

# Perform cleanup
for file in "${FILES[@]}"; do
    echo "Processing $file..."
    # Remove type declarations matching the problematic types
    sed -i -E "/type (NotificationChannel|NotificationTemplate|EventCallback|SecurityRule|SecurityPolicy|PerformanceThreshold|ReportSchedule|ReportDistributor|NetworkFunctionManager|NetworkFunction) /d" "$file"
done

echo "✅ Type duplicate cleanup complete\!"
go fmt ./pkg/oran/o1/...
go vet ./pkg/oran/o1/...
