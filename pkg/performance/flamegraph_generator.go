
package performance



import (

	"bytes"

	"context"

	"fmt"

	"io"

	"os"

	"os/exec"

	"path/filepath"

	"runtime"

	"runtime/pprof"

	"sort"

	"strings"

	"sync"

	"time"



	"github.com/google/pprof/profile"



	"k8s.io/klog/v2"

)



// FlameGraphGenerator generates flamegraphs from profiling data.

type FlameGraphGenerator struct {

	profileDir   string

	outputDir    string

	beforeData   map[string]*ProfileData

	afterData    map[string]*ProfileData

	comparisons  map[string]*FlameGraphComparisonResult

	mu           sync.RWMutex

	svgGenerator SVGGenerator

}



// ProfileData represents collected profile data.

type ProfileData struct {

	Type         string

	Timestamp    time.Time

	ProfilePath  string

	FlameGraph   string

	TotalSamples int64

	HotSpots     []HotSpot

	Metrics      ProfileMetrics

}



// ProfileMetrics contains profile-specific metrics.

type ProfileMetrics struct {

	CPUCycles      int64

	HeapAlloc      uint64

	HeapInUse      uint64

	GoroutineCount int

	BlockTime      time.Duration

	MutexWaitTime  time.Duration

	SampleRate     int

}



// FlameGraphComparisonResult represents before/after comparison for flame graphs.

type FlameGraphComparisonResult struct {

	Type           string

	BeforeProfile  *ProfileData

	AfterProfile   *ProfileData

	Improvement    float64

	HotSpotChanges []HotSpotChange

	Summary        string

	FlameGraphDiff string

}



// HotSpotChange represents changes in hot spots.

type HotSpotChange struct {

	Function      string

	BeforeSamples int64

	AfterSamples  int64

	BeforePercent float64

	AfterPercent  float64

	ChangePercent float64

	Status        string // "eliminated", "reduced", "increased", "new"

}



// SVGGenerator generates SVG flamegraphs.

type SVGGenerator struct {

	width      int

	height     int

	cellHeight int

	minWidth   float64

	colors     map[string]string

}



// NewFlameGraphGenerator creates a new flamegraph generator.

func NewFlameGraphGenerator(profileDir, outputDir string) *FlameGraphGenerator {

	return &FlameGraphGenerator{

		profileDir:  profileDir,

		outputDir:   outputDir,

		beforeData:  make(map[string]*ProfileData),

		afterData:   make(map[string]*ProfileData),

		comparisons: make(map[string]*FlameGraphComparisonResult),

		svgGenerator: SVGGenerator{

			width:      1200,

			height:     800,

			cellHeight: 16,

			minWidth:   0.1,

			colors: map[string]string{

				"hot":       "#ff0000",

				"warm":      "#ff8800",

				"normal":    "#ffcc00",

				"cool":      "#88ff00",

				"cold":      "#00ff00",

				"improved":  "#00aa00",

				"regressed": "#aa0000",

			},

		},

	}

}



// CaptureBeforeProfile captures profile data before optimization.

func (fg *FlameGraphGenerator) CaptureBeforeProfile(ctx context.Context, profileType string, duration time.Duration) (*ProfileData, error) {

	fg.mu.Lock()

	defer fg.mu.Unlock()



	klog.Infof("Capturing BEFORE profile for %s", profileType)



	profileData := &ProfileData{

		Type:      profileType,

		Timestamp: time.Now(),

	}



	switch profileType {

	case "cpu":

		path, err := fg.captureCPUProfile(ctx, duration, "before")

		if err != nil {

			return nil, fmt.Errorf("failed to capture CPU profile: %w", err)

		}

		profileData.ProfilePath = path



	case "memory":

		path, err := fg.captureMemoryProfile("before")

		if err != nil {

			return nil, fmt.Errorf("failed to capture memory profile: %w", err)

		}

		profileData.ProfilePath = path



	case "goroutine":

		path, err := fg.captureGoroutineProfile("before")

		if err != nil {

			return nil, fmt.Errorf("failed to capture goroutine profile: %w", err)

		}

		profileData.ProfilePath = path



	case "block":

		path, err := fg.captureBlockProfile(ctx, duration, "before")

		if err != nil {

			return nil, fmt.Errorf("failed to capture block profile: %w", err)

		}

		profileData.ProfilePath = path



	case "mutex":

		path, err := fg.captureMutexProfile(ctx, duration, "before")

		if err != nil {

			return nil, fmt.Errorf("failed to capture mutex profile: %w", err)

		}

		profileData.ProfilePath = path



	default:

		return nil, fmt.Errorf("unsupported profile type: %s", profileType)

	}



	// Analyze profile and extract hot spots.

	if err := fg.analyzeProfile(profileData); err != nil {

		klog.Errorf("Failed to analyze profile: %v", err)

	}



	// Generate flamegraph.

	flamegraph, err := fg.generateFlameGraph(profileData.ProfilePath, fmt.Sprintf("%s_before", profileType))

	if err != nil {

		klog.Errorf("Failed to generate flamegraph: %v", err)

	} else {

		profileData.FlameGraph = flamegraph

	}



	fg.beforeData[profileType] = profileData

	return profileData, nil

}



// CaptureAfterProfile captures profile data after optimization.

func (fg *FlameGraphGenerator) CaptureAfterProfile(ctx context.Context, profileType string, duration time.Duration) (*ProfileData, error) {

	fg.mu.Lock()

	defer fg.mu.Unlock()



	klog.Infof("Capturing AFTER profile for %s", profileType)



	profileData := &ProfileData{

		Type:      profileType,

		Timestamp: time.Now(),

	}



	switch profileType {

	case "cpu":

		path, err := fg.captureCPUProfile(ctx, duration, "after")

		if err != nil {

			return nil, fmt.Errorf("failed to capture CPU profile: %w", err)

		}

		profileData.ProfilePath = path



	case "memory":

		path, err := fg.captureMemoryProfile("after")

		if err != nil {

			return nil, fmt.Errorf("failed to capture memory profile: %w", err)

		}

		profileData.ProfilePath = path



	case "goroutine":

		path, err := fg.captureGoroutineProfile("after")

		if err != nil {

			return nil, fmt.Errorf("failed to capture goroutine profile: %w", err)

		}

		profileData.ProfilePath = path



	case "block":

		path, err := fg.captureBlockProfile(ctx, duration, "after")

		if err != nil {

			return nil, fmt.Errorf("failed to capture block profile: %w", err)

		}

		profileData.ProfilePath = path



	case "mutex":

		path, err := fg.captureMutexProfile(ctx, duration, "after")

		if err != nil {

			return nil, fmt.Errorf("failed to capture mutex profile: %w", err)

		}

		profileData.ProfilePath = path



	default:

		return nil, fmt.Errorf("unsupported profile type: %s", profileType)

	}



	// Analyze profile and extract hot spots.

	if err := fg.analyzeProfile(profileData); err != nil {

		klog.Errorf("Failed to analyze profile: %v", err)

	}



	// Generate flamegraph.

	flamegraph, err := fg.generateFlameGraph(profileData.ProfilePath, fmt.Sprintf("%s_after", profileType))

	if err != nil {

		klog.Errorf("Failed to generate flamegraph: %v", err)

	} else {

		profileData.FlameGraph = flamegraph

	}



	fg.afterData[profileType] = profileData

	return profileData, nil

}



// GenerateComparison generates before/after comparison.

func (fg *FlameGraphGenerator) GenerateComparison(profileType string) (*FlameGraphComparisonResult, error) {

	fg.mu.RLock()

	defer fg.mu.RUnlock()



	before, hasBefore := fg.beforeData[profileType]

	after, hasAfter := fg.afterData[profileType]



	if !hasBefore || !hasAfter {

		return nil, fmt.Errorf("missing before/after data for %s", profileType)

	}



	comparison := &FlameGraphComparisonResult{

		Type:          profileType,

		BeforeProfile: before,

		AfterProfile:  after,

	}



	// Calculate improvement.

	comparison.Improvement = fg.calculateImprovement(before, after)



	// Analyze hot spot changes.

	comparison.HotSpotChanges = fg.analyzeHotSpotChanges(before, after)



	// Generate comparison flamegraph.

	diffGraph, err := fg.generateDiffFlameGraph(before.ProfilePath, after.ProfilePath, profileType)

	if err != nil {

		klog.Errorf("Failed to generate diff flamegraph: %v", err)

	} else {

		comparison.FlameGraphDiff = diffGraph

	}



	// Generate summary.

	comparison.Summary = fg.generateComparisonSummary(comparison)



	fg.comparisons[profileType] = comparison

	return comparison, nil

}



// captureCPUProfile captures CPU profile.

func (fg *FlameGraphGenerator) captureCPUProfile(ctx context.Context, duration time.Duration, suffix string) (string, error) {

	filename := filepath.Join(fg.profileDir, fmt.Sprintf("cpu_%s_%d.prof", suffix, time.Now().Unix()))

	file, err := os.Create(filename)

	if err != nil {

		return "", err

	}

	defer file.Close()



	// Start CPU profiling.

	if err := pprof.StartCPUProfile(file); err != nil {

		return "", err

	}



	// Wait for duration or context cancellation.

	select {

	case <-time.After(duration):

	case <-ctx.Done():

	}



	pprof.StopCPUProfile()

	klog.Infof("CPU profile saved to %s", filename)

	return filename, nil

}



// captureMemoryProfile captures memory profile.

func (fg *FlameGraphGenerator) captureMemoryProfile(suffix string) (string, error) {

	runtime.GC() // Force GC for accurate profile



	filename := filepath.Join(fg.profileDir, fmt.Sprintf("mem_%s_%d.prof", suffix, time.Now().Unix()))

	file, err := os.Create(filename)

	if err != nil {

		return "", err

	}

	defer file.Close()



	if err := pprof.WriteHeapProfile(file); err != nil {

		return "", err

	}



	klog.Infof("Memory profile saved to %s", filename)

	return filename, nil

}



// captureGoroutineProfile captures goroutine profile.

func (fg *FlameGraphGenerator) captureGoroutineProfile(suffix string) (string, error) {

	filename := filepath.Join(fg.profileDir, fmt.Sprintf("goroutine_%s_%d.prof", suffix, time.Now().Unix()))

	file, err := os.Create(filename)

	if err != nil {

		return "", err

	}

	defer file.Close()



	profile := pprof.Lookup("goroutine")

	if err := profile.WriteTo(file, 2); err != nil {

		return "", err

	}



	klog.Infof("Goroutine profile saved to %s", filename)

	return filename, nil

}



// captureBlockProfile captures block profile.

func (fg *FlameGraphGenerator) captureBlockProfile(ctx context.Context, duration time.Duration, suffix string) (string, error) {

	runtime.SetBlockProfileRate(1) // Profile everything

	defer runtime.SetBlockProfileRate(0)



	// Wait for duration to collect samples.

	select {

	case <-time.After(duration):

	case <-ctx.Done():

	}



	filename := filepath.Join(fg.profileDir, fmt.Sprintf("block_%s_%d.prof", suffix, time.Now().Unix()))

	file, err := os.Create(filename)

	if err != nil {

		return "", err

	}

	defer file.Close()



	profile := pprof.Lookup("block")

	if err := profile.WriteTo(file, 0); err != nil {

		return "", err

	}



	klog.Infof("Block profile saved to %s", filename)

	return filename, nil

}



// captureMutexProfile captures mutex profile.

func (fg *FlameGraphGenerator) captureMutexProfile(ctx context.Context, duration time.Duration, suffix string) (string, error) {

	runtime.SetMutexProfileFraction(1) // Profile everything

	defer runtime.SetMutexProfileFraction(0)



	// Wait for duration to collect samples.

	select {

	case <-time.After(duration):

	case <-ctx.Done():

	}



	filename := filepath.Join(fg.profileDir, fmt.Sprintf("mutex_%s_%d.prof", suffix, time.Now().Unix()))

	file, err := os.Create(filename)

	if err != nil {

		return "", err

	}

	defer file.Close()



	profile := pprof.Lookup("mutex")

	if err := profile.WriteTo(file, 0); err != nil {

		return "", err

	}



	klog.Infof("Mutex profile saved to %s", filename)

	return filename, nil

}



// analyzeProfile analyzes a profile and extracts hot spots.

func (fg *FlameGraphGenerator) analyzeProfile(profileData *ProfileData) error {

	prof, err := profile.Parse(openFile(profileData.ProfilePath))

	if err != nil {

		return fmt.Errorf("failed to parse profile: %w", err)

	}



	// Calculate total samples.

	var totalSamples int64

	for _, sample := range prof.Sample {

		totalSamples += sample.Value[0]

	}

	profileData.TotalSamples = totalSamples



	// Extract hot spots.

	hotSpots := make(map[string]int64)

	for _, sample := range prof.Sample {

		for _, location := range sample.Location {

			for _, line := range location.Line {

				funcName := line.Function.Name

				hotSpots[funcName] += sample.Value[0]

			}

		}

	}



	// Sort and convert to HotSpot slice.

	var spots []HotSpot

	for funcName, samples := range hotSpots {

		percentage := float64(samples) / float64(totalSamples) * 100

		if percentage > 0.1 { // Only include functions > 0.1%

			spots = append(spots, HotSpot{

				Function:   funcName,

				Samples:    int(samples),

				Percentage: percentage,

			})

		}

	}



	// Sort by percentage descending.

	sort.Slice(spots, func(i, j int) bool {

		return spots[i].Percentage > spots[j].Percentage

	})



	// Keep top 20 hot spots.

	if len(spots) > 20 {

		spots = spots[:20]

	}



	profileData.HotSpots = spots

	return nil

}



// generateFlameGraph generates an SVG flamegraph.

func (fg *FlameGraphGenerator) generateFlameGraph(profilePath, name string) (string, error) {

	outputPath := filepath.Join(fg.outputDir, fmt.Sprintf("%s_flamegraph.svg", name))



	// Try to use go tool pprof to generate flamegraph.

	cmd := exec.Command("go", "tool", "pprof", "-svg", "-output", outputPath, profilePath)

	if output, err := cmd.CombinedOutput(); err != nil {

		// Fallback to custom SVG generation.

		klog.Warningf("pprof SVG generation failed: %v, output: %s. Using custom generator.", err, output)

		return fg.generateCustomFlameGraph(profilePath, outputPath, name)

	}



	klog.Infof("Flamegraph generated at %s", outputPath)

	return outputPath, nil

}



// generateCustomFlameGraph generates a custom SVG flamegraph.

func (fg *FlameGraphGenerator) generateCustomFlameGraph(profilePath, outputPath, name string) (string, error) {

	prof, err := profile.Parse(openFile(profilePath))

	if err != nil {

		return "", fmt.Errorf("failed to parse profile: %w", err)

	}



	// Build call tree.

	tree := fg.buildCallTree(prof)



	// Generate SVG.

	svg := fg.svgGenerator.GenerateFlameGraph(tree, name)



	// Write to file.

	if err := os.WriteFile(outputPath, []byte(svg), 0o640); err != nil {

		return "", fmt.Errorf("failed to write SVG: %w", err)

	}



	return outputPath, nil

}



// buildCallTree builds a call tree from profile data.

func (fg *FlameGraphGenerator) buildCallTree(prof *profile.Profile) *CallNode {

	root := &CallNode{

		Name:     "root",

		Value:    0,

		Children: make(map[string]*CallNode),

	}



	for _, sample := range prof.Sample {

		value := sample.Value[0]

		current := root



		// Build path from leaf to root (reverse order).

		path := make([]string, 0)

		for i := len(sample.Location) - 1; i >= 0; i-- {

			location := sample.Location[i]

			for _, line := range location.Line {

				path = append(path, line.Function.Name)

			}

		}



		// Add path to tree.

		for _, funcName := range path {

			if child, exists := current.Children[funcName]; exists {

				child.Value += value

				current = child

			} else {

				newNode := &CallNode{

					Name:     funcName,

					Value:    value,

					Children: make(map[string]*CallNode),

				}

				current.Children[funcName] = newNode

				current = newNode

			}

		}

		root.Value += value

	}



	return root

}



// CallNode represents a node in the call tree.

type CallNode struct {

	Name     string

	Value    int64

	Children map[string]*CallNode

}



// generateDiffFlameGraph generates a differential flamegraph.

func (fg *FlameGraphGenerator) generateDiffFlameGraph(beforePath, afterPath, profileType string) (string, error) {

	outputPath := filepath.Join(fg.outputDir, fmt.Sprintf("%s_diff_flamegraph.svg", profileType))



	// Parse both profiles.

	beforeProf, err := profile.Parse(openFile(beforePath))

	if err != nil {

		return "", fmt.Errorf("failed to parse before profile: %w", err)

	}



	afterProf, err := profile.Parse(openFile(afterPath))

	if err != nil {

		return "", fmt.Errorf("failed to parse after profile: %w", err)

	}



	// Build diff tree.

	beforeTree := fg.buildCallTree(beforeProf)

	afterTree := fg.buildCallTree(afterProf)

	diffTree := fg.buildDiffTree(beforeTree, afterTree)



	// Generate diff SVG.

	svg := fg.svgGenerator.GenerateDiffFlameGraph(diffTree, profileType)



	// Write to file.

	if err := os.WriteFile(outputPath, []byte(svg), 0o640); err != nil {

		return "", fmt.Errorf("failed to write diff SVG: %w", err)

	}



	klog.Infof("Diff flamegraph generated at %s", outputPath)

	return outputPath, nil

}



// buildDiffTree builds a differential call tree.

func (fg *FlameGraphGenerator) buildDiffTree(before, after *CallNode) *DiffNode {

	diff := &DiffNode{

		Name:        after.Name,

		BeforeValue: before.Value,

		AfterValue:  after.Value,

		Children:    make(map[string]*DiffNode),

	}



	// Process all nodes that exist in after.

	for name, afterChild := range after.Children {

		var beforeChild *CallNode

		if before != nil {

			beforeChild = before.Children[name]

		}

		if beforeChild == nil {

			beforeChild = &CallNode{Name: name, Value: 0, Children: make(map[string]*CallNode)}

		}

		diff.Children[name] = fg.buildDiffTree(beforeChild, afterChild)

	}



	// Process nodes that only exist in before (eliminated hot spots).

	if before != nil {

		for name, beforeChild := range before.Children {

			if _, exists := after.Children[name]; !exists {

				afterChild := &CallNode{Name: name, Value: 0, Children: make(map[string]*CallNode)}

				diff.Children[name] = fg.buildDiffTree(beforeChild, afterChild)

			}

		}

	}



	return diff

}



// DiffNode represents a node in the differential call tree.

type DiffNode struct {

	Name        string

	BeforeValue int64

	AfterValue  int64

	Children    map[string]*DiffNode

}



// calculateImprovement calculates the improvement percentage.

func (fg *FlameGraphGenerator) calculateImprovement(before, after *ProfileData) float64 {

	if before.TotalSamples == 0 {

		return 0

	}



	reduction := float64(before.TotalSamples-after.TotalSamples) / float64(before.TotalSamples) * 100

	return reduction

}



// analyzeHotSpotChanges analyzes changes in hot spots.

func (fg *FlameGraphGenerator) analyzeHotSpotChanges(before, after *ProfileData) []HotSpotChange {

	changes := make([]HotSpotChange, 0)



	// Create maps for easy lookup.

	beforeMap := make(map[string]HotSpot)

	for _, spot := range before.HotSpots {

		beforeMap[spot.Function] = spot

	}



	afterMap := make(map[string]HotSpot)

	for _, spot := range after.HotSpots {

		afterMap[spot.Function] = spot

	}



	// Check all functions that existed before.

	for funcName, beforeSpot := range beforeMap {

		change := HotSpotChange{

			Function:      funcName,

			BeforeSamples: int64(beforeSpot.Samples),

			BeforePercent: beforeSpot.Percentage,

		}



		if afterSpot, exists := afterMap[funcName]; exists {

			change.AfterSamples = int64(afterSpot.Samples)

			change.AfterPercent = afterSpot.Percentage

			change.ChangePercent = afterSpot.Percentage - beforeSpot.Percentage



			if change.ChangePercent < -50 {

				change.Status = "eliminated"

			} else if change.ChangePercent < 0 {

				change.Status = "reduced"

			} else if change.ChangePercent > 0 {

				change.Status = "increased"

			}

		} else {

			change.Status = "eliminated"

			change.ChangePercent = -100

		}



		changes = append(changes, change)

	}



	// Check for new hot spots.

	for funcName, afterSpot := range afterMap {

		if _, exists := beforeMap[funcName]; !exists {

			changes = append(changes, HotSpotChange{

				Function:      funcName,

				AfterSamples:  int64(afterSpot.Samples),

				AfterPercent:  afterSpot.Percentage,

				ChangePercent: 100,

				Status:        "new",

			})

		}

	}



	// Sort by absolute change percentage.

	sort.Slice(changes, func(i, j int) bool {

		return abs(changes[i].ChangePercent) > abs(changes[j].ChangePercent)

	})



	return changes

}



// generateComparisonSummary generates a comparison summary.

func (fg *FlameGraphGenerator) generateComparisonSummary(comparison *FlameGraphComparisonResult) string {

	var summary strings.Builder



	summary.WriteString(fmt.Sprintf("=== %s Profile Comparison ===\n", comparison.Type))

	summary.WriteString(fmt.Sprintf("Overall Improvement: %.2f%%\n", comparison.Improvement))

	summary.WriteString(fmt.Sprintf("Total Samples: %d -> %d (%.2f%% reduction)\n",

		comparison.BeforeProfile.TotalSamples,

		comparison.AfterProfile.TotalSamples,

		float64(comparison.BeforeProfile.TotalSamples-comparison.AfterProfile.TotalSamples)/float64(comparison.BeforeProfile.TotalSamples)*100))



	summary.WriteString("\nTop Hot Spot Changes:\n")

	for i, change := range comparison.HotSpotChanges {

		if i >= 10 {

			break

		}



		symbol := "→"

		if change.Status == "eliminated" {

			symbol = "✓"

		} else if change.Status == "reduced" {

			symbol = "↓"

		} else if change.Status == "increased" {

			symbol = "↑"

		} else if change.Status == "new" {

			symbol = "+"

		}



		summary.WriteString(fmt.Sprintf("  %s %s: %.2f%% -> %.2f%% (%.2f%% change)\n",

			symbol, truncateString(change.Function, 50),

			change.BeforePercent, change.AfterPercent, change.ChangePercent))

	}



	// Count eliminated hot spots.

	eliminated := 0

	reduced := 0

	for _, change := range comparison.HotSpotChanges {

		if change.Status == "eliminated" {

			eliminated++

		} else if change.Status == "reduced" {

			reduced++

		}

	}



	summary.WriteString(fmt.Sprintf("\nHot Spots Eliminated: %d\n", eliminated))

	summary.WriteString(fmt.Sprintf("Hot Spots Reduced: %d\n", reduced))



	return summary.String()

}



// GenerateFlameGraph generates an SVG flamegraph from a call tree.

func (sg *SVGGenerator) GenerateFlameGraph(root *CallNode, title string) string {

	var svg strings.Builder



	// SVG header.

	svg.WriteString(fmt.Sprintf(`<?xml version="1.0" standalone="no"?>

<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">

<svg version="1.1" width="%d" height="%d" xmlns="http://www.w3.org/2000/svg">

<rect x="0" y="0" width="%d" height="%d" fill="white"/>

<text x="%d" y="20" font-size="16" font-family="Verdana" text-anchor="middle">%s</text>

`, sg.width, sg.height, sg.width, sg.height, sg.width/2, title))



	// Draw flame graph.

	sg.drawNode(&svg, root, 0, 30, float64(sg.width), 0)



	svg.WriteString("</svg>")

	return svg.String()

}



// GenerateDiffFlameGraph generates a differential flamegraph.

func (sg *SVGGenerator) GenerateDiffFlameGraph(root *DiffNode, title string) string {

	var svg strings.Builder



	// SVG header.

	svg.WriteString(fmt.Sprintf(`<?xml version="1.0" standalone="no"?>

<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">

<svg version="1.1" width="%d" height="%d" xmlns="http://www.w3.org/2000/svg">

<rect x="0" y="0" width="%d" height="%d" fill="white"/>

<text x="%d" y="20" font-size="16" font-family="Verdana" text-anchor="middle">%s Differential FlameGraph</text>

`, sg.width, sg.height, sg.width, sg.height, sg.width/2, title))



	// Draw diff flame graph.

	sg.drawDiffNode(&svg, root, 0, 30, float64(sg.width), 0)



	// Legend.

	svg.WriteString(fmt.Sprintf(`

<text x="10" y="%d" font-size="12" font-family="Verdana">Legend:</text>

<rect x="10" y="%d" width="20" height="10" fill="%s"/>

<text x="35" y="%d" font-size="10" font-family="Verdana">Improved (reduced samples)</text>

<rect x="200" y="%d" width="20" height="10" fill="%s"/>

<text x="225" y="%d" font-size="10" font-family="Verdana">Regressed (increased samples)</text>

`, sg.height-40, sg.height-30, sg.colors["improved"], sg.height-22, sg.height-30, sg.colors["regressed"], sg.height-22))



	svg.WriteString("</svg>")

	return svg.String()

}



// drawNode draws a node in the flame graph.

func (sg *SVGGenerator) drawNode(svg *strings.Builder, node *CallNode, x, y, width float64, depth int) {

	if width < sg.minWidth || node == nil {

		return

	}



	// Calculate color based on heat.

	color := sg.getHeatColor(node.Value, node.Value) // Simplified for now



	// Draw rectangle.

	fmt.Fprintf(svg, `<rect x="%.1f" y="%.1f" width="%.1f" height="%d" fill="%s" stroke="white"/>`,

		x, y, width, sg.cellHeight, color)



	// Draw text if wide enough.

	if width > 50 {

		text := truncateString(node.Name, int(width/7))

		fmt.Fprintf(svg, `<text x="%.1f" y="%.1f" font-size="10" font-family="Verdana" text-anchor="middle">%s</text>`,

			x+width/2, y+float64(sg.cellHeight)/2+3, text)

	}



	// Draw children.

	if len(node.Children) > 0 {

		childY := y + float64(sg.cellHeight)

		childX := x



		// Calculate total value for width distribution.

		var totalValue int64

		for _, child := range node.Children {

			totalValue += child.Value

		}



		// Draw each child.

		for _, child := range node.Children {

			childWidth := (float64(child.Value) / float64(totalValue)) * width

			sg.drawNode(svg, child, childX, childY, childWidth, depth+1)

			childX += childWidth

		}

	}

}



// drawDiffNode draws a differential node.

func (sg *SVGGenerator) drawDiffNode(svg *strings.Builder, node *DiffNode, x, y, width float64, depth int) {

	if width < sg.minWidth || node == nil {

		return

	}



	// Calculate color based on improvement/regression.

	var color string

	if node.AfterValue < node.BeforeValue {

		color = sg.colors["improved"]

	} else if node.AfterValue > node.BeforeValue {

		color = sg.colors["regressed"]

	} else {

		color = sg.colors["normal"]

	}



	// Draw rectangle.

	fmt.Fprintf(svg, `<rect x="%.1f" y="%.1f" width="%.1f" height="%d" fill="%s" stroke="white"/>`,

		x, y, width, sg.cellHeight, color)



	// Draw text if wide enough.

	if width > 50 {

		text := truncateString(node.Name, int(width/7))

		change := float64(node.AfterValue-node.BeforeValue) / float64(node.BeforeValue) * 100

		if node.BeforeValue == 0 && node.AfterValue > 0 {

			change = 100

		}

		fmt.Fprintf(svg, `<text x="%.1f" y="%.1f" font-size="10" font-family="Verdana" text-anchor="middle">%s (%.1f%%)</text>`,

			x+width/2, y+float64(sg.cellHeight)/2+3, text, change)

	}



	// Draw children.

	if len(node.Children) > 0 {

		childY := y + float64(sg.cellHeight)

		childX := x



		// Use after values for width distribution.

		var totalValue int64

		for _, child := range node.Children {

			if child.AfterValue > 0 {

				totalValue += child.AfterValue

			} else {

				totalValue += child.BeforeValue // For eliminated functions

			}

		}



		// Draw each child.

		for _, child := range node.Children {

			var childValue int64

			if child.AfterValue > 0 {

				childValue = child.AfterValue

			} else {

				childValue = child.BeforeValue

			}

			childWidth := (float64(childValue) / float64(totalValue)) * width

			sg.drawDiffNode(svg, child, childX, childY, childWidth, depth+1)

			childX += childWidth

		}

	}

}



// getHeatColor returns a color based on heat value.

func (sg *SVGGenerator) getHeatColor(value, maxValue int64) string {

	ratio := float64(value) / float64(maxValue)

	if ratio > 0.8 {

		return sg.colors["hot"]

	} else if ratio > 0.6 {

		return sg.colors["warm"]

	} else if ratio > 0.4 {

		return sg.colors["normal"]

	} else if ratio > 0.2 {

		return sg.colors["cool"]

	}

	return sg.colors["cold"]

}



// Helper functions.



func openFile(path string) io.Reader {

	file, err := os.Open(path)

	if err != nil {

		return bytes.NewReader([]byte{})

	}

	return file

}



func truncateString(s string, maxLen int) string {

	if len(s) <= maxLen {

		return s

	}

	return s[:maxLen-3] + "..."

}



func abs(x float64) float64 {

	if x < 0 {

		return -x

	}

	return x

}



// GetComparisonReport generates a comprehensive comparison report.

func (fg *FlameGraphGenerator) GetComparisonReport() string {

	fg.mu.RLock()

	defer fg.mu.RUnlock()



	var report strings.Builder

	report.WriteString("=== PERFORMANCE OPTIMIZATION FLAMEGRAPH REPORT ===\n\n")



	for _, comparison := range fg.comparisons {

		report.WriteString(comparison.Summary)

		report.WriteString("\nFlameGraphs:\n")

		report.WriteString(fmt.Sprintf("  Before: %s\n", comparison.BeforeProfile.FlameGraph))

		report.WriteString(fmt.Sprintf("  After: %s\n", comparison.AfterProfile.FlameGraph))

		report.WriteString(fmt.Sprintf("  Diff: %s\n", comparison.FlameGraphDiff))

		report.WriteString("\n" + strings.Repeat("-", 60) + "\n\n")

	}



	return report.String()

}

