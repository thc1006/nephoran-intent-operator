package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// TechnicalDebtMonitor provides comprehensive technical debt tracking and analysis
type TechnicalDebtMonitor struct {
	ProjectPath   string                    `json:"project_path"`
	Timestamp     time.Time                 `json:"timestamp"`
	TotalDebt     DebtSummary              `json:"total_debt"`
	DebtByCategory map[string]DebtCategory  `json:"debt_by_category"`
	DebtByFile     map[string]FileDebtInfo  `json:"debt_by_file"`
	DebtByPackage  map[string]PackageDebt   `json:"debt_by_package"`
	DebtTrend      []DebtTrendPoint         `json:"debt_trend"`
	Alerts         []DebtAlert              `json:"alerts"`
	Recommendations []DebtRecommendation    `json:"recommendations"`
	Configuration  DebtConfiguration        `json:"configuration"`
	Metrics        DebtMetrics              `json:"metrics"`
}

type DebtSummary struct {
	TotalEstimatedHours    float64 `json:"total_estimated_hours"`
	TotalEstimatedMinutes  float64 `json:"total_estimated_minutes"`
	TotalIssues           int     `json:"total_issues"`
	HighPriorityIssues    int     `json:"high_priority_issues"`
	MediumPriorityIssues  int     `json:"medium_priority_issues"`
	LowPriorityIssues     int     `json:"low_priority_issues"`
	DebtRatio             float64 `json:"debt_ratio"`
	DebtIndex             float64 `json:"debt_index"`
	MaintenanceLoad       string  `json:"maintenance_load"`
	RecommendedAction     string  `json:"recommended_action"`
}

type DebtCategory struct {
	Name                string      `json:"name"`
	Description         string      `json:"description"`
	Issues              []DebtIssue `json:"issues"`
	EstimatedHours      float64     `json:"estimated_hours"`
	Priority            string      `json:"priority"`
	TrendDirection      string      `json:"trend_direction"`
	RecommendedAction   string      `json:"recommended_action"`
}

type DebtIssue struct {
	Type            string    `json:"type"`
	Category        string    `json:"category"`
	File            string    `json:"file"`
	Line            int       `json:"line"`
	Column          int       `json:"column"`
	Severity        string    `json:"severity"`
	Priority        string    `json:"priority"`
	Description     string    `json:"description"`
	Recommendation  string    `json:"recommendation"`
	EstimatedMinutes int      `json:"estimated_minutes"`
	Context         string    `json:"context"`
	Tags            []string  `json:"tags"`
	DetectedAt      time.Time `json:"detected_at"`
	LastSeen        time.Time `json:"last_seen"`
	FixComplexity   string    `json:"fix_complexity"`
	BusinessImpact  string    `json:"business_impact"`
}

type FileDebtInfo struct {
	FilePath            string      `json:"file_path"`
	TotalDebtMinutes    int         `json:"total_debt_minutes"`
	IssueCount          int         `json:"issue_count"`
	DebtDensity         float64     `json:"debt_density"`
	LinesOfCode         int         `json:"lines_of_code"`
	ComplexityScore     int         `json:"complexity_score"`
	Issues              []DebtIssue `json:"issues"`
	LastModified        time.Time   `json:"last_modified"`
	RecommendedAction   string      `json:"recommended_action"`
}

type PackageDebt struct {
	PackageName         string               `json:"package_name"`
	TotalDebtMinutes    int                  `json:"total_debt_minutes"`
	FileCount           int                  `json:"file_count"`
	IssueCount          int                  `json:"issue_count"`
	AverageDebtPerFile  float64              `json:"average_debt_per_file"`
	Files               map[string]FileDebtInfo `json:"files"`
	Categories          map[string]int       `json:"categories"`
	RecommendedAction   string               `json:"recommended_action"`
}

type DebtTrendPoint struct {
	Date                time.Time `json:"date"`
	TotalDebtMinutes    int       `json:"total_debt_minutes"`
	TotalIssues         int       `json:"total_issues"`
	NewIssues           int       `json:"new_issues"`
	ResolvedIssues      int       `json:"resolved_issues"`
	DebtRatio           float64   `json:"debt_ratio"`
}

type DebtAlert struct {
	Type        string    `json:"type"`
	Level       string    `json:"level"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	ActionRequired string `json:"action_required"`
	Threshold   string    `json:"threshold"`
	CurrentValue string   `json:"current_value"`
	DetectedAt  time.Time `json:"detected_at"`
	Files       []string  `json:"files,omitempty"`
}

type DebtRecommendation struct {
	Category    string   `json:"category"`
	Priority    string   `json:"priority"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Benefits    string   `json:"benefits"`
	Effort      string   `json:"effort"`
	Timeline    string   `json:"timeline"`
	Files       []string `json:"files,omitempty"`
	EstimatedHours float64 `json:"estimated_hours"`
}

type DebtConfiguration struct {
	MaxDebtRatio                 float64              `json:"max_debt_ratio"`
	MaxComplexityPerFunction     int                  `json:"max_complexity_per_function"`
	MaxLinesPerFile              int                  `json:"max_lines_per_file"`
	MaxFunctionsPerFile          int                  `json:"max_functions_per_file"`
	MinTestCoverage              float64              `json:"min_test_coverage"`
	MinCommentRatio              float64              `json:"min_comment_ratio"`
	MaxDuplicationPercent        float64              `json:"max_duplication_percent"`
	DebtPatterns                 map[string]int       `json:"debt_patterns"`
	SeverityWeights              map[string]int       `json:"severity_weights"`
	CategoryPriorities           map[string]string    `json:"category_priorities"`
}

type DebtMetrics struct {
	DebtVelocity        float64 `json:"debt_velocity"`        // Debt accumulation rate per day
	ResolutionRate      float64 `json:"resolution_rate"`      // Issues resolved per day
	DebtBurndown        float64 `json:"debt_burndown"`        // Time to resolve all debt (days)
	QualityGate         bool    `json:"quality_gate"`         // Whether debt is within acceptable limits
	RiskLevel          string  `json:"risk_level"`           // Overall risk level
	MaintenanceBurden  float64 `json:"maintenance_burden"`   // % of dev time spent on debt
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run technical-debt-monitor.go <project-path> [config-file] [output-file]")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  go run technical-debt-monitor.go . # Analyze current directory")
		fmt.Println("  go run technical-debt-monitor.go . debt-config.json debt-report.json")
		os.Exit(1)
	}

	projectPath := os.Args[1]
	configFile := ""
	outputFile := "technical-debt-report.json"

	if len(os.Args) > 2 {
		configFile = os.Args[2]
	}
	if len(os.Args) > 3 {
		outputFile = os.Args[3]
	}

	monitor := NewTechnicalDebtMonitor(projectPath)

	// Load configuration if provided
	if configFile != "" {
		if err := monitor.LoadConfiguration(configFile); err != nil {
			log.Printf("Warning: Could not load configuration file %s: %v", configFile, err)
		}
	}

	fmt.Printf("🔍 Technical Debt Analysis Starting...\n")
	fmt.Printf("📁 Project Path: %s\n", projectPath)
	fmt.Printf("📊 Configuration: %s\n", configFile)
	fmt.Printf("📋 Output File: %s\n\n", outputFile)

	// Run comprehensive debt analysis
	if err := monitor.AnalyzeProject(); err != nil {
		log.Fatalf("Error analyzing project: %v", err)
	}

	// Generate alerts and recommendations
	monitor.GenerateAlerts()
	monitor.GenerateRecommendations()

	// Calculate metrics
	monitor.CalculateMetrics()

	// Save results
	if err := monitor.SaveReport(outputFile); err != nil {
		log.Fatalf("Error saving report: %v", err)
	}

	// Print summary
	monitor.PrintSummary()

	// Exit with appropriate code
	if monitor.Metrics.QualityGate {
		fmt.Printf("\n✅ Technical debt is within acceptable limits\n")
	} else {
		fmt.Printf("\n⚠️  Technical debt requires attention\n")
		if monitor.Metrics.RiskLevel == "HIGH" {
			os.Exit(1)
		}
	}
}

func NewTechnicalDebtMonitor(projectPath string) *TechnicalDebtMonitor {
	return &TechnicalDebtMonitor{
		ProjectPath:    projectPath,
		Timestamp:      time.Now(),
		DebtByCategory: make(map[string]DebtCategory),
		DebtByFile:     make(map[string]FileDebtInfo),
		DebtByPackage:  make(map[string]PackageDebt),
		Configuration: DebtConfiguration{
			MaxDebtRatio:             0.3,  // 30% max debt ratio
			MaxComplexityPerFunction: 15,   // Max cyclomatic complexity
			MaxLinesPerFile:          500,  // Max lines per file
			MaxFunctionsPerFile:      20,   // Max functions per file
			MinTestCoverage:          0.8,  // 80% minimum test coverage
			MinCommentRatio:          0.15, // 15% minimum comment ratio
			MaxDuplicationPercent:    5.0,  // 5% max code duplication
			DebtPatterns: map[string]int{
				"TODO":       5,   // 5 minutes per TODO
				"FIXME":      15,  // 15 minutes per FIXME
				"HACK":       30,  // 30 minutes per HACK
				"XXX":        20,  // 20 minutes per XXX
				"BUG":        45,  // 45 minutes per BUG
				"DEPRECATED": 60,  // 60 minutes per deprecated usage
			},
			SeverityWeights: map[string]int{
				"HIGH":   3,
				"MEDIUM": 2,
				"LOW":    1,
			},
			CategoryPriorities: map[string]string{
				"security":      "HIGH",
				"performance":   "HIGH",
				"maintainability": "MEDIUM",
				"complexity":    "MEDIUM",
				"documentation": "LOW",
				"style":        "LOW",
			},
		},
	}
}

func (monitor *TechnicalDebtMonitor) LoadConfiguration(configFile string) error {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &monitor.Configuration)
}

func (monitor *TechnicalDebtMonitor) AnalyzeProject() error {
	fmt.Printf("🔍 Scanning project for technical debt indicators...\n")

	err := filepath.Walk(monitor.ProjectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if monitor.shouldSkipFile(path) {
			return nil
		}

		if strings.HasSuffix(path, ".go") {
			return monitor.analyzeGoFile(path)
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Analyze package-level debt
	monitor.analyzePackageDebt()

	// Calculate total debt
	monitor.calculateTotalDebt()

	return nil
}

func (monitor *TechnicalDebtMonitor) shouldSkipFile(path string) bool {
	skipPatterns := []string{
		"vendor/",
		".git/",
		"node_modules/",
		"testdata/",
		"pb.go",
		"_generated.go",
		"zz_generated",
		".pb.go",
	}

	for _, pattern := range skipPatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}

	return false
}

func (monitor *TechnicalDebtMonitor) analyzeGoFile(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	fileInfo := FileDebtInfo{
		FilePath: filePath,
		Issues:   []DebtIssue{},
	}

	// Analyze file content
	lines := strings.Split(string(content), "\n")
	fileInfo.LinesOfCode = len(lines)

	// Parse AST for structural analysis
	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		// If parsing fails, do basic analysis
		return monitor.analyzeFileBasic(filePath, string(content), &fileInfo)
	}

	// Analyze AST
	monitor.analyzeAST(astFile, fset, &fileInfo)

	// Analyze comments and code patterns
	monitor.analyzeCodePatterns(lines, &fileInfo)

	// Calculate debt density
	if fileInfo.LinesOfCode > 0 {
		fileInfo.DebtDensity = float64(fileInfo.TotalDebtMinutes) / float64(fileInfo.LinesOfCode)
	}

	// Get file modification time
	if stat, err := os.Stat(filePath); err == nil {
		fileInfo.LastModified = stat.ModTime()
	}

	// Generate recommendations for this file
	monitor.generateFileRecommendations(&fileInfo)

	monitor.DebtByFile[filePath] = fileInfo
	return nil
}

func (monitor *TechnicalDebtMonitor) analyzeAST(astFile *ast.File, fset *token.FileSet, fileInfo *FileDebtInfo) {
	functionCount := 0
	totalComplexity := 0

	ast.Inspect(astFile, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			functionCount++
			complexity := monitor.calculateCyclomaticComplexity(node)
			totalComplexity += complexity

			if complexity > monitor.Configuration.MaxComplexityPerFunction {
				pos := fset.Position(node.Pos())
				issue := DebtIssue{
					Type:             "high_complexity",
					Category:         "complexity",
					File:             fileInfo.FilePath,
					Line:             pos.Line,
					Column:           pos.Column,
					Severity:         "MEDIUM",
					Priority:         "MEDIUM",
					Description:      fmt.Sprintf("Function %s has high cyclomatic complexity (%d > %d)", node.Name.Name, complexity, monitor.Configuration.MaxComplexityPerFunction),
					Recommendation:   "Break down complex function into smaller, more focused functions",
					EstimatedMinutes: complexity * 10, // 10 minutes per excess complexity point
					Context:          node.Name.Name,
					Tags:             []string{"complexity", "maintainability"},
					DetectedAt:       time.Now(),
					LastSeen:         time.Now(),
					FixComplexity:    "MEDIUM",
					BusinessImpact:   "Reduces code maintainability and increases bug risk",
				}

				fileInfo.Issues = append(fileInfo.Issues, issue)
				fileInfo.TotalDebtMinutes += issue.EstimatedMinutes
			}
		}
		return true
	})

	fileInfo.ComplexityScore = totalComplexity

	// Check for large file issues
	if fileInfo.LinesOfCode > monitor.Configuration.MaxLinesPerFile {
		issue := DebtIssue{
			Type:             "large_file",
			Category:         "maintainability",
			File:             fileInfo.FilePath,
			Line:             1,
			Severity:         "MEDIUM",
			Priority:         "MEDIUM",
			Description:      fmt.Sprintf("File is too large (%d lines > %d)", fileInfo.LinesOfCode, monitor.Configuration.MaxLinesPerFile),
			Recommendation:   "Split large file into smaller, more cohesive modules",
			EstimatedMinutes: (fileInfo.LinesOfCode - monitor.Configuration.MaxLinesPerFile) / 10, // 1 minute per 10 excess lines
			Tags:             []string{"maintainability", "structure"},
			DetectedAt:       time.Now(),
			LastSeen:         time.Now(),
			FixComplexity:    "HIGH",
			BusinessImpact:   "Makes code harder to navigate and understand",
		}

		fileInfo.Issues = append(fileInfo.Issues, issue)
		fileInfo.TotalDebtMinutes += issue.EstimatedMinutes
	}

	// Check for too many functions per file
	if functionCount > monitor.Configuration.MaxFunctionsPerFile {
		issue := DebtIssue{
			Type:             "too_many_functions",
			Category:         "maintainability",
			File:             fileInfo.FilePath,
			Line:             1,
			Severity:         "LOW",
			Priority:         "LOW",
			Description:      fmt.Sprintf("File has too many functions (%d > %d)", functionCount, monitor.Configuration.MaxFunctionsPerFile),
			Recommendation:   "Consider splitting functions across multiple files or packages",
			EstimatedMinutes: (functionCount - monitor.Configuration.MaxFunctionsPerFile) * 5,
			Tags:             []string{"maintainability", "structure"},
			DetectedAt:       time.Now(),
			LastSeen:         time.Now(),
			FixComplexity:    "MEDIUM",
			BusinessImpact:   "Reduces code organization and readability",
		}

		fileInfo.Issues = append(fileInfo.Issues, issue)
		fileInfo.TotalDebtMinutes += issue.EstimatedMinutes
	}
}

func (monitor *TechnicalDebtMonitor) calculateCyclomaticComplexity(fn *ast.FuncDecl) int {
	complexity := 1

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

func (monitor *TechnicalDebtMonitor) analyzeCodePatterns(lines []string, fileInfo *FileDebtInfo) {
	commentLines := 0
	blankLines := 0

	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		
		if line == "" {
			blankLines++
			continue
		}

		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") {
			commentLines++
		}

		// Check for debt patterns
		for pattern, minutes := range monitor.Configuration.DebtPatterns {
			if monitor.containsDebtPattern(line, pattern) {
				severity := "LOW"
				priority := "LOW"

				switch pattern {
				case "BUG", "FIXME":
					severity = "HIGH"
					priority = "HIGH"
				case "HACK", "XXX":
					severity = "MEDIUM"
					priority = "MEDIUM"
				}

				issue := DebtIssue{
					Type:             strings.ToLower(pattern),
					Category:         "maintenance",
					File:             fileInfo.FilePath,
					Line:             lineNum + 1,
					Severity:         severity,
					Priority:         priority,
					Description:      fmt.Sprintf("Found %s comment: %s", pattern, strings.TrimSpace(line)),
					Recommendation:   monitor.getPatternRecommendation(pattern),
					EstimatedMinutes: minutes,
					Context:          line,
					Tags:             []string{"maintenance", "comments"},
					DetectedAt:       time.Now(),
					LastSeen:         time.Now(),
					FixComplexity:    "LOW",
					BusinessImpact:   monitor.getPatternBusinessImpact(pattern),
				}

				fileInfo.Issues = append(fileInfo.Issues, issue)
				fileInfo.TotalDebtMinutes += issue.EstimatedMinutes
			}
		}

		// Check for other problematic patterns
		monitor.checkForAntiPatterns(line, lineNum+1, fileInfo)
	}

	// Check comment ratio
	commentRatio := float64(commentLines) / float64(len(lines))
	if commentRatio < monitor.Configuration.MinCommentRatio {
		issue := DebtIssue{
			Type:             "low_documentation",
			Category:         "documentation",
			File:             fileInfo.FilePath,
			Line:             1,
			Severity:         "LOW",
			Priority:         "LOW",
			Description:      fmt.Sprintf("Low comment ratio (%.1f%% < %.1f%%)", commentRatio*100, monitor.Configuration.MinCommentRatio*100),
			Recommendation:   "Add comments to explain complex logic and public APIs",
			EstimatedMinutes: int((monitor.Configuration.MinCommentRatio - commentRatio) * float64(len(lines)) * 0.5),
			Tags:             []string{"documentation", "maintainability"},
			DetectedAt:       time.Now(),
			LastSeen:         time.Now(),
			FixComplexity:    "LOW",
			BusinessImpact:   "Reduces code understandability for new team members",
		}

		fileInfo.Issues = append(fileInfo.Issues, issue)
		fileInfo.TotalDebtMinutes += issue.EstimatedMinutes
	}

	fileInfo.IssueCount = len(fileInfo.Issues)
}

func (monitor *TechnicalDebtMonitor) containsDebtPattern(line, pattern string) bool {
	// Case-insensitive search for debt patterns
	upperLine := strings.ToUpper(line)
	upperPattern := strings.ToUpper(pattern)
	
	// Look for pattern followed by colon or whitespace
	patterns := []string{
		upperPattern + ":",
		upperPattern + " ",
		upperPattern + "\t",
		"// " + upperPattern,
		"/* " + upperPattern,
	}

	for _, p := range patterns {
		if strings.Contains(upperLine, p) {
			return true
		}
	}

	return false
}

func (monitor *TechnicalDebtMonitor) checkForAntiPatterns(line string, lineNum int, fileInfo *FileDebtInfo) {
	antiPatterns := map[string]string{
		"panic(":           "Avoid using panic in production code",
		"fmt.Print":        "Use structured logging instead of fmt.Print",
		"time.Sleep":       "Avoid time.Sleep in production code, use proper synchronization",
		"//nolint":         "Linting suppressions should be minimized",
		"interface{}":      "Use specific types instead of empty interface when possible",
		"_ = ":             "Explicitly handle errors instead of discarding them",
	}

	for pattern, recommendation := range antiPatterns {
		if strings.Contains(line, pattern) {
			issue := DebtIssue{
				Type:             "anti_pattern",
				Category:         "quality",
				File:             fileInfo.FilePath,
				Line:             lineNum,
				Severity:         "MEDIUM",
				Priority:         "MEDIUM",
				Description:      fmt.Sprintf("Potential anti-pattern detected: %s", pattern),
				Recommendation:   recommendation,
				EstimatedMinutes: 15,
				Context:          line,
				Tags:             []string{"quality", "best_practices"},
				DetectedAt:       time.Now(),
				LastSeen:         time.Now(),
				FixComplexity:    "MEDIUM",
				BusinessImpact:   "May impact code quality and maintainability",
			}

			fileInfo.Issues = append(fileInfo.Issues, issue)
			fileInfo.TotalDebtMinutes += issue.EstimatedMinutes
		}
	}
}

func (monitor *TechnicalDebtMonitor) getPatternRecommendation(pattern string) string {
	recommendations := map[string]string{
		"TODO":       "Complete the TODO item or create a proper issue/ticket",
		"FIXME":      "Fix the identified issue as soon as possible",
		"HACK":       "Replace hack with a proper solution",
		"XXX":        "Address the marked concern or uncertainty",
		"BUG":        "Fix the bug immediately",
		"DEPRECATED": "Update to use the current recommended approach",
	}

	if rec, exists := recommendations[pattern]; exists {
		return rec
	}

	return "Address this technical debt item"
}

func (monitor *TechnicalDebtMonitor) getPatternBusinessImpact(pattern string) string {
	impacts := map[string]string{
		"TODO":       "Represents incomplete functionality",
		"FIXME":      "Indicates known defects that may cause issues",
		"HACK":       "Increases maintenance risk and reduces code quality",
		"XXX":        "Highlights areas of uncertainty that need attention",
		"BUG":        "Represents actual defects affecting functionality",
		"DEPRECATED": "Using deprecated features may cause future compatibility issues",
	}

	if impact, exists := impacts[pattern]; exists {
		return impact
	}

	return "May impact code quality and maintainability"
}

func (monitor *TechnicalDebtMonitor) analyzeFileBasic(filePath, content string, fileInfo *FileDebtInfo) error {
	lines := strings.Split(content, "\n")
	fileInfo.LinesOfCode = len(lines)

	for lineNum, line := range lines {
		for pattern, minutes := range monitor.Configuration.DebtPatterns {
			if monitor.containsDebtPattern(line, pattern) {
				issue := DebtIssue{
					Type:             strings.ToLower(pattern),
					Category:         "maintenance",
					File:             fileInfo.FilePath,
					Line:             lineNum + 1,
					Severity:         "MEDIUM",
					Description:      fmt.Sprintf("Found %s comment", pattern),
					EstimatedMinutes: minutes,
					DetectedAt:       time.Now(),
					LastSeen:         time.Now(),
				}

				fileInfo.Issues = append(fileInfo.Issues, issue)
				fileInfo.TotalDebtMinutes += issue.EstimatedMinutes
			}
		}
	}

	fileInfo.IssueCount = len(fileInfo.Issues)
	return nil
}

func (monitor *TechnicalDebtMonitor) generateFileRecommendations(fileInfo *FileDebtInfo) {
	if fileInfo.TotalDebtMinutes > 60 {
		fileInfo.RecommendedAction = "High debt file - prioritize refactoring"
	} else if fileInfo.TotalDebtMinutes > 30 {
		fileInfo.RecommendedAction = "Moderate debt - include in next maintenance sprint"
	} else {
		fileInfo.RecommendedAction = "Low debt - address during regular development"
	}
}

func (monitor *TechnicalDebtMonitor) analyzePackageDebt() {
	packageMap := make(map[string]*PackageDebt)

	for filePath, fileInfo := range monitor.DebtByFile {
		packageName := monitor.getPackageName(filePath)
		
		if _, exists := packageMap[packageName]; !exists {
			packageMap[packageName] = &PackageDebt{
				PackageName: packageName,
				Files:       make(map[string]FileDebtInfo),
				Categories:  make(map[string]int),
			}
		}

		pkg := packageMap[packageName]
		pkg.Files[filePath] = fileInfo
		pkg.FileCount++
		pkg.TotalDebtMinutes += fileInfo.TotalDebtMinutes
		pkg.IssueCount += fileInfo.IssueCount

		// Categorize issues
		for _, issue := range fileInfo.Issues {
			pkg.Categories[issue.Category]++
		}
	}

	// Calculate package-level metrics and recommendations
	for _, pkg := range packageMap {
		if pkg.FileCount > 0 {
			pkg.AverageDebtPerFile = float64(pkg.TotalDebtMinutes) / float64(pkg.FileCount)
		}

		if pkg.TotalDebtMinutes > 300 {
			pkg.RecommendedAction = "Critical package - requires immediate attention"
		} else if pkg.TotalDebtMinutes > 120 {
			pkg.RecommendedAction = "High debt package - schedule refactoring"
		} else {
			pkg.RecommendedAction = "Manageable debt level"
		}

		monitor.DebtByPackage[pkg.PackageName] = *pkg
	}
}

func (monitor *TechnicalDebtMonitor) getPackageName(filePath string) string {
	rel, err := filepath.Rel(monitor.ProjectPath, filePath)
	if err != nil {
		return filepath.Dir(filePath)
	}
	
	dir := filepath.Dir(rel)
	if dir == "." {
		return "root"
	}
	
	return strings.ReplaceAll(dir, string(filepath.Separator), "/")
}

func (monitor *TechnicalDebtMonitor) calculateTotalDebt() {
	totalMinutes := 0
	totalIssues := 0
	highPriority := 0
	mediumPriority := 0
	lowPriority := 0

	categories := make(map[string]*DebtCategory)

	for _, fileInfo := range monitor.DebtByFile {
		totalMinutes += fileInfo.TotalDebtMinutes
		totalIssues += fileInfo.IssueCount

		for _, issue := range fileInfo.Issues {
			switch issue.Priority {
			case "HIGH":
				highPriority++
			case "MEDIUM":
				mediumPriority++
			default:
				lowPriority++
			}

			// Group by category
			if _, exists := categories[issue.Category]; !exists {
				categories[issue.Category] = &DebtCategory{
					Name:   issue.Category,
					Issues: []DebtIssue{},
				}
			}
			categories[issue.Category].Issues = append(categories[issue.Category].Issues, issue)
			categories[issue.Category].EstimatedHours += float64(issue.EstimatedMinutes) / 60
		}
	}

	// Calculate category-level information
	for name, category := range categories {
		category.Name = name
		category.Description = monitor.getCategoryDescription(name)
		category.Priority = monitor.Configuration.CategoryPriorities[name]
		category.TrendDirection = "stable" // Would need historical data
		category.RecommendedAction = monitor.getCategoryRecommendation(name, category)
		
		monitor.DebtByCategory[name] = *category
	}

	monitor.TotalDebt = DebtSummary{
		TotalEstimatedMinutes: float64(totalMinutes),
		TotalEstimatedHours:   float64(totalMinutes) / 60,
		TotalIssues:          totalIssues,
		HighPriorityIssues:   highPriority,
		MediumPriorityIssues: mediumPriority,
		LowPriorityIssues:    lowPriority,
		DebtRatio:           monitor.calculateDebtRatio(),
		DebtIndex:           monitor.calculateDebtIndex(),
	}

	// Set maintenance load and recommended action
	if monitor.TotalDebt.DebtRatio > 0.4 {
		monitor.TotalDebt.MaintenanceLoad = "CRITICAL"
		monitor.TotalDebt.RecommendedAction = "Stop feature development and focus on debt reduction"
	} else if monitor.TotalDebt.DebtRatio > 0.25 {
		monitor.TotalDebt.MaintenanceLoad = "HIGH"
		monitor.TotalDebt.RecommendedAction = "Allocate significant time to debt reduction"
	} else if monitor.TotalDebt.DebtRatio > 0.15 {
		monitor.TotalDebt.MaintenanceLoad = "MODERATE"
		monitor.TotalDebt.RecommendedAction = "Include debt reduction in regular sprints"
	} else {
		monitor.TotalDebt.MaintenanceLoad = "LOW"
		monitor.TotalDebt.RecommendedAction = "Monitor and address debt opportunistically"
	}
}

func (monitor *TechnicalDebtMonitor) calculateDebtRatio() float64 {
	totalLOC := 0
	for _, fileInfo := range monitor.DebtByFile {
		totalLOC += fileInfo.LinesOfCode
	}

	if totalLOC == 0 {
		return 0
	}

	return float64(monitor.TotalDebt.TotalEstimatedMinutes) / float64(totalLOC) * 100
}

func (monitor *TechnicalDebtMonitor) calculateDebtIndex() float64 {
	// Simplified debt index calculation
	if monitor.TotalDebt.TotalIssues == 0 {
		return 0
	}

	weightedIssues := float64(monitor.TotalDebt.HighPriorityIssues*3 + 
		monitor.TotalDebt.MediumPriorityIssues*2 + 
		monitor.TotalDebt.LowPriorityIssues*1)

	return weightedIssues / float64(monitor.TotalDebt.TotalIssues) * 10
}

func (monitor *TechnicalDebtMonitor) getCategoryDescription(category string) string {
	descriptions := map[string]string{
		"complexity":      "Issues related to high cyclomatic complexity and code structure",
		"maintainability": "Issues affecting long-term code maintainability",
		"documentation":   "Missing or inadequate code documentation",
		"maintenance":     "TODO, FIXME, and other maintenance markers",
		"quality":         "Code quality and best practice violations",
		"security":        "Potential security vulnerabilities and concerns",
		"performance":     "Performance-related technical debt",
	}

	if desc, exists := descriptions[category]; exists {
		return desc
	}

	return "Technical debt issues in this category"
}

func (monitor *TechnicalDebtMonitor) getCategoryRecommendation(category string, cat *DebtCategory) string {
	if len(cat.Issues) == 0 {
		return "No issues in this category"
	}

	recommendations := map[string]string{
		"complexity":      "Refactor complex functions and improve code structure",
		"maintainability": "Improve code organization and reduce coupling",
		"documentation":   "Add comprehensive documentation for public APIs",
		"maintenance":     "Address TODO/FIXME items and maintenance markers",
		"quality":         "Follow coding standards and best practices",
		"security":        "Address security vulnerabilities immediately",
		"performance":     "Profile and optimize performance bottlenecks",
	}

	if rec, exists := recommendations[category]; exists {
		return rec
	}

	return "Address issues in this category according to priority"
}

func (monitor *TechnicalDebtMonitor) GenerateAlerts() {
	var alerts []DebtAlert

	// High debt ratio alert
	if monitor.TotalDebt.DebtRatio > monitor.Configuration.MaxDebtRatio {
		alerts = append(alerts, DebtAlert{
			Type:         "debt_ratio",
			Level:        "HIGH",
			Title:        "Technical Debt Ratio Exceeded",
			Description:  fmt.Sprintf("Technical debt ratio (%.2f) exceeds threshold (%.2f)", monitor.TotalDebt.DebtRatio, monitor.Configuration.MaxDebtRatio),
			ActionRequired: "Immediate debt reduction required",
			Threshold:    fmt.Sprintf("%.2f", monitor.Configuration.MaxDebtRatio),
			CurrentValue: fmt.Sprintf("%.2f", monitor.TotalDebt.DebtRatio),
			DetectedAt:   time.Now(),
		})
	}

	// High priority issues alert
	if monitor.TotalDebt.HighPriorityIssues > 10 {
		alerts = append(alerts, DebtAlert{
			Type:         "high_priority_issues",
			Level:        "MEDIUM",
			Title:        "Too Many High Priority Issues",
			Description:  fmt.Sprintf("Found %d high priority technical debt issues", monitor.TotalDebt.HighPriorityIssues),
			ActionRequired: "Address high priority issues in next sprint",
			Threshold:    "10",
			CurrentValue: fmt.Sprintf("%d", monitor.TotalDebt.HighPriorityIssues),
			DetectedAt:   time.Now(),
		})
	}

	// Large files alert
	var largeFiles []string
	for filePath, fileInfo := range monitor.DebtByFile {
		if fileInfo.LinesOfCode > monitor.Configuration.MaxLinesPerFile {
			largeFiles = append(largeFiles, filePath)
		}
	}

	if len(largeFiles) > 0 {
		alerts = append(alerts, DebtAlert{
			Type:         "large_files",
			Level:        "MEDIUM",
			Title:        "Large Files Detected",
			Description:  fmt.Sprintf("Found %d files exceeding size threshold", len(largeFiles)),
			ActionRequired: "Consider splitting large files",
			Threshold:    fmt.Sprintf("%d lines", monitor.Configuration.MaxLinesPerFile),
			CurrentValue: fmt.Sprintf("%d files", len(largeFiles)),
			DetectedAt:   time.Now(),
			Files:        largeFiles,
		})
	}

	monitor.Alerts = alerts
}

func (monitor *TechnicalDebtMonitor) GenerateRecommendations() {
	var recommendations []DebtRecommendation

	// Sort categories by estimated hours
	type categoryWithHours struct {
		name  string
		hours float64
	}

	var sortedCategories []categoryWithHours
	for name, category := range monitor.DebtByCategory {
		sortedCategories = append(sortedCategories, categoryWithHours{
			name:  name,
			hours: category.EstimatedHours,
		})
	}

	sort.Slice(sortedCategories, func(i, j int) bool {
		return sortedCategories[i].hours > sortedCategories[j].hours
	})

	// Generate recommendations for top categories
	for i, cat := range sortedCategories {
		if i >= 5 { // Top 5 categories
			break
		}

		category := monitor.DebtByCategory[cat.name]
		
		priority := "MEDIUM"
		if category.EstimatedHours > 8 {
			priority = "HIGH"
		} else if category.EstimatedHours < 2 {
			priority = "LOW"
		}

		timeline := "2-4 weeks"
		if category.EstimatedHours > 16 {
			timeline = "1-2 months"
		} else if category.EstimatedHours < 4 {
			timeline = "1 week"
		}

		recommendations = append(recommendations, DebtRecommendation{
			Category:       cat.name,
			Priority:       priority,
			Title:          fmt.Sprintf("Address %s Issues", strings.Title(cat.name)),
			Description:    category.Description,
			Benefits:       monitor.getCategoryBenefits(cat.name),
			Effort:         monitor.getEffortEstimate(category.EstimatedHours),
			Timeline:       timeline,
			EstimatedHours: category.EstimatedHours,
		})
	}

	// Add specific recommendations for high-impact areas
	if monitor.TotalDebt.DebtRatio > 0.3 {
		recommendations = append(recommendations, DebtRecommendation{
			Category:    "overall",
			Priority:    "HIGH",
			Title:       "Comprehensive Debt Reduction Initiative",
			Description: "Technical debt has reached critical levels requiring systematic reduction",
			Benefits:    "Improved development velocity, reduced bug rates, easier maintenance",
			Effort:      "HIGH",
			Timeline:    "2-3 months",
			EstimatedHours: monitor.TotalDebt.TotalEstimatedHours,
		})
	}

	monitor.Recommendations = recommendations
}

func (monitor *TechnicalDebtMonitor) getCategoryBenefits(category string) string {
	benefits := map[string]string{
		"complexity":      "Improved code readability and reduced bug risk",
		"maintainability": "Faster feature development and easier code changes",
		"documentation":   "Better developer onboarding and code understanding",
		"maintenance":     "Reduced future maintenance burden",
		"quality":         "Higher code quality and adherence to best practices",
		"security":        "Improved security posture and risk reduction",
		"performance":     "Better system performance and user experience",
	}

	if benefit, exists := benefits[category]; exists {
		return benefit
	}

	return "Improved overall code quality"
}

func (monitor *TechnicalDebtMonitor) getEffortEstimate(hours float64) string {
	if hours > 40 {
		return "VERY HIGH"
	} else if hours > 16 {
		return "HIGH"
	} else if hours > 8 {
		return "MEDIUM"
	} else if hours > 2 {
		return "LOW"
	} else {
		return "VERY LOW"
	}
}

func (monitor *TechnicalDebtMonitor) CalculateMetrics() {
	// Debt velocity (would need historical data)
	monitor.Metrics.DebtVelocity = 0 // Placeholder

	// Resolution rate (would need historical data)
	monitor.Metrics.ResolutionRate = 0 // Placeholder

	// Debt burndown time
	if monitor.Metrics.ResolutionRate > 0 {
		monitor.Metrics.DebtBurndown = monitor.TotalDebt.TotalEstimatedHours / monitor.Metrics.ResolutionRate
	} else {
		monitor.Metrics.DebtBurndown = monitor.TotalDebt.TotalEstimatedHours / 8 // Assume 8 hours per day
	}

	// Quality gate
	monitor.Metrics.QualityGate = monitor.TotalDebt.DebtRatio <= monitor.Configuration.MaxDebtRatio

	// Risk level
	if monitor.TotalDebt.DebtRatio > 0.4 {
		monitor.Metrics.RiskLevel = "CRITICAL"
	} else if monitor.TotalDebt.DebtRatio > 0.25 {
		monitor.Metrics.RiskLevel = "HIGH"
	} else if monitor.TotalDebt.DebtRatio > 0.15 {
		monitor.Metrics.RiskLevel = "MEDIUM"
	} else {
		monitor.Metrics.RiskLevel = "LOW"
	}

	// Maintenance burden (simplified)
	monitor.Metrics.MaintenanceBurden = monitor.TotalDebt.DebtRatio / 100 * 40 // Rough estimate
}

func (monitor *TechnicalDebtMonitor) SaveReport(filename string) error {
	data, err := json.MarshalIndent(monitor, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

func (monitor *TechnicalDebtMonitor) PrintSummary() {
	fmt.Printf("\n" + strings.Repeat("=", 80) + "\n")
	fmt.Printf("⚡ TECHNICAL DEBT ANALYSIS SUMMARY\n")
	fmt.Printf(strings.Repeat("=", 80) + "\n\n")

	fmt.Printf("📊 Overall Assessment:\n")
	fmt.Printf("  • Total Debt: %.1f hours (%.0f minutes)\n", 
		monitor.TotalDebt.TotalEstimatedHours, monitor.TotalDebt.TotalEstimatedMinutes)
	fmt.Printf("  • Debt Ratio: %.2f (threshold: %.2f)\n", 
		monitor.TotalDebt.DebtRatio, monitor.Configuration.MaxDebtRatio)
	fmt.Printf("  • Risk Level: %s\n", monitor.Metrics.RiskLevel)
	fmt.Printf("  • Maintenance Load: %s\n\n", monitor.TotalDebt.MaintenanceLoad)

	fmt.Printf("🎯 Issue Breakdown:\n")
	fmt.Printf("  • Total Issues: %d\n", monitor.TotalDebt.TotalIssues)
	fmt.Printf("  • High Priority: %d\n", monitor.TotalDebt.HighPriorityIssues)
	fmt.Printf("  • Medium Priority: %d\n", monitor.TotalDebt.MediumPriorityIssues)
	fmt.Printf("  • Low Priority: %d\n\n", monitor.TotalDebt.LowPriorityIssues)

	// Top debt categories
	if len(monitor.DebtByCategory) > 0 {
		fmt.Printf("📂 Top Debt Categories:\n")
		
		type categoryPair struct {
			name  string
			hours float64
		}
		
		var pairs []categoryPair
		for name, category := range monitor.DebtByCategory {
			pairs = append(pairs, categoryPair{name, category.EstimatedHours})
		}
		
		sort.Slice(pairs, func(i, j int) bool {
			return pairs[i].hours > pairs[j].hours
		})
		
		for i, pair := range pairs {
			if i >= 5 {
				break
			}
			fmt.Printf("  %d. %s: %.1f hours\n", i+1, strings.Title(pair.name), pair.hours)
		}
		fmt.Println()
	}

	// Alerts
	if len(monitor.Alerts) > 0 {
		fmt.Printf("🚨 Active Alerts:\n")
		for i, alert := range monitor.Alerts {
			fmt.Printf("  %d. [%s] %s\n", i+1, alert.Level, alert.Title)
			fmt.Printf("     %s\n", alert.Description)
		}
		fmt.Println()
	}

	// Recommendations
	if len(monitor.Recommendations) > 0 {
		fmt.Printf("💡 Top Recommendations:\n")
		for i, rec := range monitor.Recommendations {
			if i >= 3 {
				break
			}
			fmt.Printf("  %d. [%s] %s (%.1f hours)\n", i+1, rec.Priority, rec.Title, rec.EstimatedHours)
			fmt.Printf("     %s\n", rec.Description)
		}
		fmt.Println()
	}

	fmt.Printf("🎯 Recommended Action: %s\n", monitor.TotalDebt.RecommendedAction)
	
	if monitor.Metrics.QualityGate {
		fmt.Printf("✅ Quality gate: PASSED\n")
	} else {
		fmt.Printf("❌ Quality gate: FAILED\n")
	}
	
	fmt.Printf("\nGenerated: %s\n", monitor.Timestamp.Format("2006-01-02 15:04:05"))
}