//go:build stub

/*

Copyright 2025.



Licensed under the Apache License, Version 2.0 (the "License");

you may not use this file except in compliance with the License.

You may obtain a copy of the License at



    http://www.apache.org/licenses/LICENSE-2.0



Unless required by applicable law or agreed to in writing, software

distributed under the License is distributed on an "AS IS" BASIS,

WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.

See the License for the specific language governing permissions and

limitations under the License.

*/

package dependencies

import (
	"time"
)

// convertUsageDataToDataPoints converts UsageData slice to UsageDataPoint slice.

func convertUsageDataToDataPoints(usageData []*UsageData) []UsageDataPoint {
	dataPoints := make([]UsageDataPoint, len(usageData))

	for i, data := range usageData {
		dataPoints[i] = UsageDataPoint{
			PackageName: data.Package.Name,

			Usage: data.Usage,

			Timestamp: data.Timestamp,

			Metadata: make(map[string]interface{}),
		}
	}

	return dataPoints
}

// Helper methods for usage analysis in analyzer.go.

// AnalyzePatterns analyzes usage patterns from usage data.

func (a *UsageAnalyzer) AnalyzePatterns(usageData []UsageDataPoint) []*UsagePattern {
	patterns := make([]*UsagePattern, 0)

	// Stub implementation - analyze patterns in usage data.

	if len(usageData) > 0 {

		pattern := &UsagePattern{
			Type: "daily",

			Frequency: int64(24 * time.Hour),

			Strength: 0.8,

			Description: "Daily usage pattern detected",

			Metadata: make(map[string]interface{}),
		}

		patterns = append(patterns, pattern)

	}

	return patterns
}

// calculateTotalUsage calculates total usage from usage data.

func (a *dependencyAnalyzer) calculateTotalUsage(usageData []UsageDataPoint) int64 {
	var total int64 = 0

	for _, data := range usageData {
		total += data.Usage
	}

	return total
}

// calculateAverageUsage calculates average usage from usage data.

func (a *dependencyAnalyzer) calculateAverageUsage(usageData []UsageDataPoint) float64 {
	if len(usageData) == 0 {
		return 0.0
	}

	total := a.calculateTotalUsage(usageData)

	return float64(total) / float64(len(usageData))
}

// calculateUsageGrowthRate calculates usage growth rate over time period.

func (a *dependencyAnalyzer) calculateUsageGrowthRate(usageData []UsageDataPoint, timeRange *TimeRange) float64 {
	if len(usageData) < 2 || timeRange == nil {
		return 0.0
	}

	// Simple growth rate calculation.

	firstPeriod := usageData[0].Usage

	lastPeriod := usageData[len(usageData)-1].Usage

	if firstPeriod == 0 {
		return 0.0
	}

	return float64(lastPeriod-firstPeriod) / float64(firstPeriod)
}

// rankPackagesByUsage ranks packages by their usage.

func (a *dependencyAnalyzer) rankPackagesByUsage(usageData []UsageDataPoint) []*UsageRanking {
	rankings := make([]*UsageRanking, 0)

	// Group usage by package and create rankings.

	packageUsage := make(map[string]int64)

	for _, data := range usageData {
		packageUsage[data.PackageName] += data.Usage
	}

	// Create ranking objects.

	for packageName, usage := range packageUsage {

		ranking := &UsageRanking{
			PackageName: packageName,

			Usage: usage,

			Rank: 1, // Simplified ranking

			Score: float64(usage),

			Metadata: make(map[string]interface{}),
		}

		rankings = append(rankings, ranking)

	}

	return rankings
}

// detectTrendingPackages detects packages with trending usage.

func (a *dependencyAnalyzer) detectTrendingPackages(usageData []UsageDataPoint) []*TrendingPackage {
	trending := make([]*TrendingPackage, 0)

	// Group data by package and analyze trends.

	packageData := make(map[string][]UsageDataPoint)

	for _, data := range usageData {
		packageData[data.PackageName] = append(packageData[data.PackageName], data)
	}

	for packageName, data := range packageData {
		if len(data) >= 2 {

			// Check if usage is trending up.

			recent := data[len(data)-1].Usage

			older := data[0].Usage

			if recent > older {

				trendingPkg := &TrendingPackage{
					PackageName: packageName,

					TrendStrength: float64(recent-older) / float64(older),

					Direction: TrendDirectionUp,

					Confidence: 0.8,

					Metadata: make(map[string]interface{}),
				}

				trending = append(trending, trendingPkg)

			}

		}
	}

	return trending
}

// findUnusedPackages finds packages that are not being used.

func (a *dependencyAnalyzer) findUnusedPackages(packages []*PackageReference, usageData []UsageDataPoint) []*UnusedPackage {
	unused := make([]*UnusedPackage, 0)

	// Create a set of used package names.

	usedPackages := make(map[string]bool)

	for _, data := range usageData {
		if data.Usage > 0 {
			usedPackages[data.PackageName] = true
		}
	}

	// Find packages that are not used.

	for _, pkg := range packages {
		if !usedPackages[pkg.Name] {

			unusedPkg := &UnusedPackage{
				Package: pkg,

				LastUsed: time.Now().Add(-30 * 24 * time.Hour), // 30 days ago

				UnusedDuration: 30 * 24 * time.Hour,

				Reason: "No usage detected",

				Impact: "low",

				Metadata: make(map[string]interface{}),
			}

			unused = append(unused, unusedPkg)

		}
	}

	return unused
}

// findUnderutilizedPackages finds packages that are underutilized.

func (a *dependencyAnalyzer) findUnderutilizedPackages(usageData []UsageDataPoint) []*UnderutilizedPackage {
	underutilized := make([]*UnderutilizedPackage, 0)

	// Calculate average usage.

	avgUsage := a.calculateAverageUsage(usageData)

	threshold := avgUsage * 0.1 // 10% of average usage

	// Group data by package.

	packageUsage := make(map[string]int64)

	for _, data := range usageData {
		packageUsage[data.PackageName] += data.Usage
	}

	// Find underutilized packages.

	for packageName, usage := range packageUsage {
		if float64(usage) < threshold && usage > 0 {

			underutilizedPkg := &UnderutilizedPackage{
				PackageName: packageName,

				CurrentUsage: usage,

				ExpectedUsage: int64(avgUsage),

				UtilizationRate: float64(usage) / avgUsage,

				Recommendation: "Consider optimizing or removing",

				Metadata: make(map[string]interface{}),
			}

			underutilized = append(underutilized, underutilizedPkg)

		}
	}

	return underutilized
}

// calculateUsageEfficiency calculates overall usage efficiency.

func (a *dependencyAnalyzer) calculateUsageEfficiency(usageData []UsageDataPoint) float64 {
	if len(usageData) == 0 {
		return 0.0
	}

	// Simple efficiency calculation based on usage variance.

	avgUsage := a.calculateAverageUsage(usageData)

	if avgUsage == 0 {
		return 0.0
	}

	// Calculate how many packages are being actively used.

	usedPackages := make(map[string]bool)

	for _, data := range usageData {
		if data.Usage > 0 {
			usedPackages[data.PackageName] = true
		}
	}

	// Efficiency is the ratio of used packages to total unique packages.

	totalPackages := make(map[string]bool)

	for _, data := range usageData {
		totalPackages[data.PackageName] = true
	}

	if len(totalPackages) == 0 {
		return 0.0
	}

	return float64(len(usedPackages)) / float64(len(totalPackages))
}

// calculateUsageWaste calculates percentage of wasted usage.

func (a *dependencyAnalyzer) calculateUsageWaste(usageData []UsageDataPoint) float64 {
	efficiency := a.calculateUsageEfficiency(usageData)

	return (1.0 - efficiency) * 100.0 // Return as percentage
}

// generateUsageOptimizations generates optimization recommendations based on usage analysis.

func (a *dependencyAnalyzer) generateUsageOptimizations(analysis *UsageAnalysis) []*UsageOptimization {
	optimizations := make([]*UsageOptimization, 0)

	// Generate optimization based on unused packages.

	if len(analysis.UnusedPackages) > 0 {

		opt := &UsageOptimization{
			Type: "remove_unused",

			Description: "Remove unused packages to reduce overhead",

			Impact: "medium",

			Effort: "low",

			EstimatedBenefit: 0.3,

			Packages: make([]*PackageReference, len(analysis.UnusedPackages)),

			Metadata: make(map[string]interface{}),
		}

		for i, pkg := range analysis.UnusedPackages {
			opt.Packages[i] = pkg.Package
		}

		optimizations = append(optimizations, opt)

	}

	// Generate optimization based on underutilized packages.

	if len(analysis.UnderutilizedPackages) > 0 {

		opt := &UsageOptimization{
			Type: "optimize_underutilized",

			Description: "Optimize or replace underutilized packages",

			Impact: "medium",

			Effort: "medium",

			EstimatedBenefit: 0.2,

			Packages: make([]*PackageReference, len(analysis.UnderutilizedPackages)),

			Metadata: make(map[string]interface{}),
		}

		for i, pkg := range analysis.UnderutilizedPackages {
			opt.Packages[i] = &PackageReference{
				Name: pkg.PackageName,
			}
		}

		optimizations = append(optimizations, opt)

	}

	return optimizations
}
