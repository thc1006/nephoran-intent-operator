//go:build quality_metrics
// +build quality_metrics

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// QualityMetrics represents comprehensive code quality metrics
type QualityMetrics struct {
	Timestamp            time.Time                 `json:"timestamp"`
	ProjectName          string                    `json:"project_name"`
	TotalFiles           int                       `json:"total_files"`
	TotalLines           int                       `json:"total_lines"`
	CodeLines            int                       `json:"code_lines"`
	TestLines            int                       `json:"test_lines"`
	CommentLines         int                       `json:"comment_lines"`
	BlankLines           int                       `json:"blank_lines"`
	Packages             int                       `json:"packages"`
	Functions            int                       `json:"functions"`
	TestFunctions        int                       `json:"test_functions"`
	CyclomaticComplexity CyclomaticMetrics         `json:"cyclomatic_complexity"`
	Duplication          DuplicationMetrics        `json:"duplication"`
	TestCoverage         TestCoverageMetrics       `json:"test_coverage"`
	TechnicalDebt        TechnicalDebtMetrics      `json:"technical_debt"`
	Maintainability      MaintainabilityMetrics    `json:"maintainability"`
	QualityScore         float64                   `json:"quality_score"`
	QualityGrade         string                    `json:"quality_grade"`
	Recommendations      []QualityRecommendation   `json:"recommendations"`
	TrendAnalysis        TrendAnalysis             `json:"trend_analysis"`
	FileMetrics          map[string]FileMetrics    `json:"file_metrics"`
	PackageMetrics       map[string]PackageMetrics `json:"package_metrics"`
}

type CyclomaticMetrics struct {
	Average                float64           `json:"average"`
	Maximum                int               `json:"maximum"`
	TotalComplexity        int               `json:"total_complexity"`
	ComplexFunctions       []ComplexFunction `json:"complex_functions"`
	ComplexityByFile       map[string]int    `json:"complexity_by_file"`
	ComplexityDistribution map[string]int    `json:"complexity_distribution"`
}

type ComplexFunction struct {
	Name       string `json:"name"`
	File       string `json:"file"`
	Line       int    `json:"line"`
	Complexity int    `json:"complexity"`
}

type DuplicationMetrics struct {
	Percentage        float64            `json:"percentage"`
	DuplicatedLines   int                `json:"duplicated_lines"`
	DuplicationBlocks []DuplicationBlock `json:"duplication_blocks"`
	DuplicationByFile map[string]int     `json:"duplication_by_file"`
}

type DuplicationBlock struct {
	File1     string `json:"file1"`
	File2     string `json:"file2"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Lines     int    `json:"lines"`
}

type TestCoverageMetrics struct {
	Percentage      float64            `json:"percentage"`
	CoveredLines    int                `json:"covered_lines"`
	TotalLines      int                `json:"total_lines"`
	CoverageByFile  map[string]float64 `json:"coverage_by_file"`
	UncoveredFiles  []string           `json:"uncovered_files"`
	TestToCodeRatio float64            `json:"test_to_code_ratio"`
}

type TechnicalDebtMetrics struct {
	DebtRatio      float64              `json:"debt_ratio"`
	DebtIndex      float64              `json:"debt_index"`
	Issues         []TechnicalDebtIssue `json:"issues"`
	DebtByCategory map[string]int       `json:"debt_by_category"`
	EstimatedHours float64              `json:"estimated_hours"`
}

type TechnicalDebtIssue struct {
	Type        string `json:"type"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Effort      int    `json:"effort_minutes"`
}

type MaintainabilityMetrics struct {
	Index              float64 `json:"index"`
	Grade              string  `json:"grade"`
	HalsteadComplexity float64 `json:"halstead_complexity"`
	CouplingIndex      float64 `json:"coupling_index"`
	CohesionIndex      float64 `json:"cohesion_index"`
	DocumentationIndex float64 `json:"documentation_index"`
}

type QualityRecommendation struct {
	Category    string   `json:"category"`
	Priority    string   `json:"priority"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Impact      string   `json:"impact"`
	Effort      string   `json:"effort"`
	Files       []string `json:"files,omitempty"`
}

type TrendAnalysis struct {
	QualityTrend    string  `json:"quality_trend"`
	CoverageTrend   string  `json:"coverage_trend"`
	ComplexityTrend string  `json:"complexity_trend"`
	DebtTrend       string  `json:"debt_trend"`
	PreviousScore   float64 `json:"previous_score"`
	ScoreChange     float64 `json:"score_change"`
}

type FileMetrics struct {
	Lines         int     `json:"lines"`
	CodeLines     int     `json:"code_lines"`
	CommentLines  int     `json:"comment_lines"`
	BlankLines    int     `json:"blank_lines"`
	Functions     int     `json:"functions"`
	Complexity    int     `json:"complexity"`
	TestCoverage  float64 `json:"test_coverage"`
	QualityScore  float64 `json:"quality_score"`
	Issues        int     `json:"issues"`
	TechnicalDebt int     `json:"technical_debt_minutes"`
}

type PackageMetrics struct {
	Files         int     `json:"files"`
	Lines         int     `json:"lines"`
	Functions     int     `json:"functions"`
	Complexity    int     `json:"complexity"`
	TestCoverage  float64 `json:"test_coverage"`
	CouplingIndex float64 `json:"coupling_index"`
	CohesionIndex float64 `json:"cohesion_index"`
	QualityScore  float64 `json:"quality_score"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run quality-metrics.go <project-path> [output-file]")
		os.Exit(1)
	}

	projectPath := os.Args[1]
	outputFile := "quality-metrics.json"
	if len(os.Args) > 2 {
		outputFile = os.Args[2]
	}

	metrics, err := analyzeProject(projectPath)
	if err != nil {
		log.Fatalf("Error analyzing project: %v", err)
	}

	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling metrics: %v", err)
	}

	err = os.WriteFile(outputFile, data, 0644)
	if err != nil {
		log.Fatalf("Error writing output file: %v", err)
	}

	// Also output summary to stdout
	printSummary(metrics)

	fmt.Printf("\nDetailed metrics written to: %s\n", outputFile)
}

func analyzeProject(projectPath string) (*QualityMetrics, error) {
	metrics := &QualityMetrics{
		Timestamp:      time.Now(),
		ProjectName:    filepath.Base(projectPath),
		FileMetrics:    make(map[string]FileMetrics),
		PackageMetrics: make(map[string]PackageMetrics),
	}

	// Initialize complexity metrics
	metrics.CyclomaticComplexity.ComplexityByFile = make(map[string]int)
	metrics.CyclomaticComplexity.ComplexityDistribution = make(map[string]int)

	// Initialize duplication metrics
	metrics.Duplication.DuplicationByFile = make(map[string]int)

	// Initialize coverage metrics
	metrics.TestCoverage.CoverageByFile = make(map[string]float64)

	// Initialize debt metrics
	metrics.TechnicalDebt.DebtByCategory = make(map[string]int)

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip non-Go files and certain directories
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		if shouldSkipFile(path) {
			return nil
		}

		fileMetrics, err := analyzeFile(path)
		if err != nil {
			log.Printf("Warning: Error analyzing file %s: %v", path, err)
			return nil
		}

		relPath, _ := filepath.Rel(projectPath, path)
		metrics.FileMetrics[relPath] = fileMetrics

		// Aggregate metrics
		metrics.TotalFiles++
		metrics.TotalLines += fileMetrics.Lines
		metrics.CodeLines += fileMetrics.CodeLines
		metrics.CommentLines += fileMetrics.CommentLines
		metrics.BlankLines += fileMetrics.BlankLines
		metrics.Functions += fileMetrics.Functions
		metrics.CyclomaticComplexity.TotalComplexity += fileMetrics.Complexity

		if strings.Contains(relPath, "_test.go") {
			metrics.TestLines += fileMetrics.CodeLines
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Calculate derived metrics
	calculateDerivedMetrics(metrics, projectPath)

	return metrics, nil
}

func analyzeFile(filePath string) (FileMetrics, error) {
	var metrics FileMetrics

	file, err := os.Open(filePath)
	if err != nil {
		return metrics, err
	}
	defer file.Close()

	// Parse file for AST analysis
	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		// If parsing fails, do basic line counting
		return analyzeFileBasic(filePath)
	}

	// Count lines
	scanner := bufio.NewScanner(file)
	file.Seek(0, 0) // Reset file pointer

	commentRegex := regexp.MustCompile(`^\s*//|^\s*/\*|\*/\s*$`)
	blankRegex := regexp.MustCompile(`^\s*$`)

	for scanner.Scan() {
		line := scanner.Text()
		metrics.Lines++

		if blankRegex.MatchString(line) {
			metrics.BlankLines++
		} else if commentRegex.MatchString(line) {
			metrics.CommentLines++
		} else {
			metrics.CodeLines++
		}
	}

	// Analyze AST for functions and complexity
	ast.Inspect(astFile, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			metrics.Functions++
			complexity := calculateCyclomaticComplexity(node)
			metrics.Complexity += complexity
		}
		return true
	})

	// Calculate quality score for this file
	metrics.QualityScore = calculateFileQualityScore(metrics)

	return metrics, nil
}

func analyzeFileBasic(filePath string) (FileMetrics, error) {
	var metrics FileMetrics

	file, err := os.Open(filePath)
	if err != nil {
		return metrics, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	commentRegex := regexp.MustCompile(`^\s*//|^\s*/\*|\*/\s*$`)
	blankRegex := regexp.MustCompile(`^\s*$`)
	funcRegex := regexp.MustCompile(`^\s*func\s+`)

	for scanner.Scan() {
		line := scanner.Text()
		metrics.Lines++

		if blankRegex.MatchString(line) {
			metrics.BlankLines++
		} else if commentRegex.MatchString(line) {
			metrics.CommentLines++
		} else {
			metrics.CodeLines++
			if funcRegex.MatchString(line) {
				metrics.Functions++
			}
		}
	}

	// Basic complexity estimation (1 per function + control structures)
	controlStructureRegex := regexp.MustCompile(`\b(if|for|while|switch|case|&&|\|\|)\b`)
	file.Seek(0, 0)
	scanner = bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		matches := controlStructureRegex.FindAllString(line, -1)
		metrics.Complexity += len(matches)
	}

	metrics.QualityScore = calculateFileQualityScore(metrics)

	return metrics, nil
}

func calculateCyclomaticComplexity(fn *ast.FuncDecl) int {
	complexity := 1 // Base complexity

	ast.Inspect(fn, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt:
			complexity++
		case *ast.RangeStmt:
			complexity++
		case *ast.ForStmt:
			complexity++
		case *ast.SwitchStmt:
			complexity++
		case *ast.CaseClause:
			complexity++
		case *ast.FuncLit:
			complexity++
		}
		return true
	})

	return complexity
}

func calculateDerivedMetrics(metrics *QualityMetrics, projectPath string) {
	// Calculate packages count
	packageSet := make(map[string]bool)
	for filePath := range metrics.FileMetrics {
		packageDir := filepath.Dir(filePath)
		packageSet[packageDir] = true
	}
	metrics.Packages = len(packageSet)

	// Calculate average complexity
	if metrics.Functions > 0 {
		metrics.CyclomaticComplexity.Average = float64(metrics.CyclomaticComplexity.TotalComplexity) / float64(metrics.Functions)
	}

	// Find maximum complexity
	for _, fileMetrics := range metrics.FileMetrics {
		if fileMetrics.Complexity > metrics.CyclomaticComplexity.Maximum {
			metrics.CyclomaticComplexity.Maximum = fileMetrics.Complexity
		}
	}

	// Calculate test coverage ratio
	if metrics.CodeLines > 0 {
		metrics.TestCoverage.TestToCodeRatio = float64(metrics.TestLines) / float64(metrics.CodeLines)
	}

	// Analyze duplication (basic implementation)
	calculateDuplicationMetrics(metrics)

	// Calculate maintainability metrics
	calculateMaintainabilityMetrics(metrics)

	// Calculate technical debt
	calculateTechnicalDebt(metrics)

	// Calculate overall quality score
	calculateQualityScore(metrics)

	// Generate recommendations
	generateRecommendations(metrics)

	// Calculate trend analysis (placeholder - would need historical data)
	calculateTrendAnalysis(metrics)
}

func calculateDuplicationMetrics(metrics *QualityMetrics) {
	// Basic duplication calculation based on similar file patterns
	// In a real implementation, this would use more sophisticated algorithms

	duplicatedLines := 0
	totalCodeLines := metrics.CodeLines

	// Simple heuristic: if files have very similar line counts and names,
	// consider some duplication
	filePairs := make(map[string][]string)
	for filePath := range metrics.FileMetrics {
		baseDir := filepath.Dir(filePath)
		baseName := strings.TrimSuffix(filepath.Base(filePath), ".go")
		key := baseDir + "/" + baseName
		filePairs[key] = append(filePairs[key], filePath)
	}

	for _, files := range filePairs {
		if len(files) > 1 {
			// Estimate duplication between similar files
			avgLines := 0
			for _, file := range files {
				avgLines += metrics.FileMetrics[file].CodeLines
			}
			avgLines /= len(files)
			duplicatedLines += avgLines / 4 // Rough estimate
		}
	}

	if totalCodeLines > 0 {
		metrics.Duplication.Percentage = float64(duplicatedLines) / float64(totalCodeLines) * 100
	}
	metrics.Duplication.DuplicatedLines = duplicatedLines
}

func calculateMaintainabilityMetrics(metrics *QualityMetrics) {
	// Calculate maintainability index using simplified formula
	// MI = max(0, (171 - 5.2 * ln(Halstead Volume) - 0.23 * Cyclomatic Complexity - 16.2 * ln(Lines of Code)) * 100 / 171)

	avgComplexity := metrics.CyclomaticComplexity.Average
	linesOfCode := float64(metrics.CodeLines)
	commentRatio := float64(metrics.CommentLines) / float64(metrics.TotalLines)

	// Simplified maintainability calculation
	complexityScore := 10 - (avgComplexity / 10 * 5) // 0-10 scale
	sizeScore := 10 - (linesOfCode / 10000 * 5)      // Penalty for large codebase
	documentationScore := commentRatio * 10          // 0-10 scale based on comment ratio

	if complexityScore < 0 {
		complexityScore = 0
	}
	if sizeScore < 0 {
		sizeScore = 0
	}
	if documentationScore > 10 {
		documentationScore = 10
	}

	metrics.Maintainability.Index = (complexityScore + sizeScore + documentationScore) / 3 * 10

	// Assign grade based on maintainability index
	if metrics.Maintainability.Index >= 85 {
		metrics.Maintainability.Grade = "A"
	} else if metrics.Maintainability.Index >= 70 {
		metrics.Maintainability.Grade = "B"
	} else if metrics.Maintainability.Index >= 55 {
		metrics.Maintainability.Grade = "C"
	} else if metrics.Maintainability.Index >= 40 {
		metrics.Maintainability.Grade = "D"
	} else {
		metrics.Maintainability.Grade = "F"
	}

	metrics.Maintainability.DocumentationIndex = commentRatio * 100
}

func calculateTechnicalDebt(metrics *QualityMetrics) {
	debtMinutes := 0.0

	// Calculate debt based on various factors
	// High complexity functions
	highComplexityPenalty := 0
	for _, fileMetrics := range metrics.FileMetrics {
		if fileMetrics.Complexity > 10 {
			highComplexityPenalty += (fileMetrics.Complexity - 10) * 30 // 30 min per excess complexity
		}
	}
	debtMinutes += float64(highComplexityPenalty)

	// Low test coverage penalty
	testCoverage := metrics.TestCoverage.TestToCodeRatio * 100
	if testCoverage < 80 {
		coverageDebt := (80 - testCoverage) * 2 // 2 minutes per missing percentage point
		debtMinutes += coverageDebt
	}

	// Large file penalty
	for _, fileMetrics := range metrics.FileMetrics {
		if fileMetrics.Lines > 500 {
			largeFileDebt := float64(fileMetrics.Lines-500) * 0.1 // 0.1 min per excess line
			debtMinutes += largeFileDebt
		}
	}

	// Low documentation penalty
	if metrics.Maintainability.DocumentationIndex < 20 {
		docDebt := (20 - metrics.Maintainability.DocumentationIndex) * 5 // 5 min per missing percentage point
		debtMinutes += docDebt
	}

	metrics.TechnicalDebt.EstimatedHours = debtMinutes / 60
	metrics.TechnicalDebt.DebtRatio = debtMinutes / float64(metrics.CodeLines) // Debt per line of code

	// Categorize debt
	metrics.TechnicalDebt.DebtByCategory["complexity"] = highComplexityPenalty
	metrics.TechnicalDebt.DebtByCategory["coverage"] = int((80 - testCoverage) * 2)
	metrics.TechnicalDebt.DebtByCategory["size"] = int(debtMinutes) - highComplexityPenalty - int((80-testCoverage)*2)
}

func calculateQualityScore(metrics *QualityMetrics) {
	// Quality score calculation (0-10 scale)
	var score float64

	// Test coverage component (30% weight)
	coverageScore := metrics.TestCoverage.TestToCodeRatio * 100
	if coverageScore > 100 {
		coverageScore = 100
	}
	score += (coverageScore / 100) * 3.0

	// Complexity component (25% weight)
	complexityScore := 10 - (metrics.CyclomaticComplexity.Average / 10 * 10)
	if complexityScore < 0 {
		complexityScore = 0
	}
	score += (complexityScore / 10) * 2.5

	// Duplication component (20% weight)
	duplicationScore := 10 - (metrics.Duplication.Percentage / 10 * 10)
	if duplicationScore < 0 {
		duplicationScore = 0
	}
	score += (duplicationScore / 10) * 2.0

	// Documentation component (15% weight)
	docScore := metrics.Maintainability.DocumentationIndex / 10
	if docScore > 10 {
		docScore = 10
	}
	score += (docScore / 10) * 1.5

	// Size/maintainability component (10% weight)
	maintScore := metrics.Maintainability.Index / 100 * 10
	score += (maintScore / 10) * 1.0

	metrics.QualityScore = score

	// Assign quality grade
	if score >= 9.0 {
		metrics.QualityGrade = "A+"
	} else if score >= 8.0 {
		metrics.QualityGrade = "A"
	} else if score >= 7.0 {
		metrics.QualityGrade = "B"
	} else if score >= 6.0 {
		metrics.QualityGrade = "C"
	} else if score >= 5.0 {
		metrics.QualityGrade = "D"
	} else {
		metrics.QualityGrade = "F"
	}
}

func generateRecommendations(metrics *QualityMetrics) {
	recommendations := []QualityRecommendation{}

	// Test coverage recommendations
	if metrics.TestCoverage.TestToCodeRatio < 0.8 {
		recommendations = append(recommendations, QualityRecommendation{
			Category:    "Testing",
			Priority:    "High",
			Title:       "Improve Test Coverage",
			Description: fmt.Sprintf("Current test coverage is %.1f%%. Target is 80%% or higher.", metrics.TestCoverage.TestToCodeRatio*100),
			Impact:      "Reduces bug risk and improves confidence in refactoring",
			Effort:      "Medium to High",
		})
	}

	// Complexity recommendations
	if metrics.CyclomaticComplexity.Average > 8 {
		recommendations = append(recommendations, QualityRecommendation{
			Category:    "Complexity",
			Priority:    "High",
			Title:       "Reduce Cyclomatic Complexity",
			Description: fmt.Sprintf("Average cyclomatic complexity is %.1f. Consider breaking down complex functions.", metrics.CyclomaticComplexity.Average),
			Impact:      "Improves code readability and maintainability",
			Effort:      "Medium",
		})
	}

	// Duplication recommendations
	if metrics.Duplication.Percentage > 5 {
		recommendations = append(recommendations, QualityRecommendation{
			Category:    "Duplication",
			Priority:    "Medium",
			Title:       "Reduce Code Duplication",
			Description: fmt.Sprintf("%.1f%% of code appears to be duplicated. Extract common functionality into shared utilities.", metrics.Duplication.Percentage),
			Impact:      "Reduces maintenance burden and bug fixing effort",
			Effort:      "Medium",
		})
	}

	// Documentation recommendations
	if metrics.Maintainability.DocumentationIndex < 20 {
		recommendations = append(recommendations, QualityRecommendation{
			Category:    "Documentation",
			Priority:    "Medium",
			Title:       "Improve Code Documentation",
			Description: fmt.Sprintf("Documentation coverage is %.1f%%. Add comments for complex functions and public APIs.", metrics.Maintainability.DocumentationIndex),
			Impact:      "Improves code understanding and onboarding",
			Effort:      "Low to Medium",
		})
	}

	// Technical debt recommendations
	if metrics.TechnicalDebt.EstimatedHours > 40 {
		recommendations = append(recommendations, QualityRecommendation{
			Category:    "Technical Debt",
			Priority:    "High",
			Title:       "Address Technical Debt",
			Description: fmt.Sprintf("Estimated technical debt is %.1f hours. Prioritize refactoring efforts.", metrics.TechnicalDebt.EstimatedHours),
			Impact:      "Reduces long-term maintenance costs and improves velocity",
			Effort:      "High",
		})
	}

	metrics.Recommendations = recommendations
}

func calculateTrendAnalysis(metrics *QualityMetrics) {
	// Placeholder for trend analysis
	// In a real implementation, this would compare with historical data
	metrics.TrendAnalysis = TrendAnalysis{
		QualityTrend:    "stable",
		CoverageTrend:   "improving",
		ComplexityTrend: "stable",
		DebtTrend:       "stable",
		PreviousScore:   metrics.QualityScore, // Placeholder
		ScoreChange:     0.0,
	}
}

func calculateFileQualityScore(fileMetrics FileMetrics) float64 {
	score := 10.0

	// Penalty for high complexity
	if fileMetrics.Complexity > fileMetrics.Functions*2 {
		score -= 2.0
	}

	// Penalty for low comment ratio
	commentRatio := float64(fileMetrics.CommentLines) / float64(fileMetrics.Lines)
	if commentRatio < 0.1 {
		score -= 1.0
	}

	// Penalty for very large files
	if fileMetrics.Lines > 500 {
		score -= 1.5
	}

	// Penalty for very few comments
	if fileMetrics.CommentLines == 0 && fileMetrics.Functions > 3 {
		score -= 1.0
	}

	if score < 0 {
		score = 0
	}

	return score
}

func shouldSkipFile(path string) bool {
	skipPatterns := []string{
		"vendor/",
		".git/",
		"testdata/",
		"pb.go",
		"generated",
		"_generated.go",
		"zz_generated",
	}

	for _, pattern := range skipPatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}

	return false
}

func printSummary(metrics *QualityMetrics) {
	fmt.Printf("\n=== CODE QUALITY METRICS SUMMARY ===\n")
	fmt.Printf("Project: %s\n", metrics.ProjectName)
	fmt.Printf("Analyzed at: %s\n\n", metrics.Timestamp.Format("2006-01-02 15:04:05"))

	fmt.Printf("📊 Overall Quality Score: %.1f/10.0 (Grade: %s)\n\n", metrics.QualityScore, metrics.QualityGrade)

	fmt.Printf("📝 Code Statistics:\n")
	fmt.Printf("  • Total Files: %d\n", metrics.TotalFiles)
	fmt.Printf("  • Total Lines: %s\n", addCommas(metrics.TotalLines))
	fmt.Printf("  • Code Lines: %s\n", addCommas(metrics.CodeLines))
	fmt.Printf("  • Test Lines: %s\n", addCommas(metrics.TestLines))
	fmt.Printf("  • Comment Lines: %s\n", addCommas(metrics.CommentLines))
	fmt.Printf("  • Packages: %d\n", metrics.Packages)
	fmt.Printf("  • Functions: %d\n\n", metrics.Functions)

	fmt.Printf("🔍 Quality Metrics:\n")
	fmt.Printf("  • Test Coverage Ratio: %.1f%%\n", metrics.TestCoverage.TestToCodeRatio*100)
	fmt.Printf("  • Avg Cyclomatic Complexity: %.1f\n", metrics.CyclomaticComplexity.Average)
	fmt.Printf("  • Code Duplication: %.1f%%\n", metrics.Duplication.Percentage)
	fmt.Printf("  • Documentation Coverage: %.1f%%\n", metrics.Maintainability.DocumentationIndex)
	fmt.Printf("  • Maintainability Index: %.1f (Grade: %s)\n\n", metrics.Maintainability.Index, metrics.Maintainability.Grade)

	fmt.Printf("⚠️  Technical Debt:\n")
	fmt.Printf("  • Estimated Effort: %.1f hours\n", metrics.TechnicalDebt.EstimatedHours)
	fmt.Printf("  • Debt Ratio: %.3f min/line\n\n", metrics.TechnicalDebt.DebtRatio)

	if len(metrics.Recommendations) > 0 {
		fmt.Printf("💡 Top Recommendations:\n")
		for i, rec := range metrics.Recommendations {
			if i >= 3 {
				break
			} // Show only top 3
			fmt.Printf("  %d. [%s] %s\n", i+1, rec.Priority, rec.Title)
			fmt.Printf("     %s\n", rec.Description)
		}
		fmt.Println()
	}

	// Quality status indicator
	if metrics.QualityScore >= 8.0 {
		fmt.Printf("✅ Code quality is EXCELLENT\n")
	} else if metrics.QualityScore >= 7.0 {
		fmt.Printf("✨ Code quality is GOOD\n")
	} else if metrics.QualityScore >= 6.0 {
		fmt.Printf("⚠️  Code quality is ACCEPTABLE but needs improvement\n")
	} else {
		fmt.Printf("❌ Code quality NEEDS ATTENTION\n")
	}
}

func addCommas(n int) string {
	str := strconv.Itoa(n)
	if len(str) <= 3 {
		return str
	}

	var result []string
	for i, r := range reverseString(str) {
		if i > 0 && i%3 == 0 {
			result = append(result, ",")
		}
		result = append(result, string(r))
	}

	return reverseString(strings.Join(result, ""))
}

func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
