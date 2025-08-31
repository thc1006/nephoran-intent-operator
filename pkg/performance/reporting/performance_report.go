package reporting

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/performance/analysis"
)

// ReportGenerator creates comprehensive performance reports.

type ReportGenerator struct {
	performanceData []analysis.PerformanceMetric
}

// NewReportGenerator initializes a report generator.

func NewReportGenerator(metrics []analysis.PerformanceMetric) *ReportGenerator {

	return &ReportGenerator{performanceData: metrics}

}

// GenerateJSONReport creates a machine-readable JSON report.

func (rg *ReportGenerator) GenerateJSONReport() ([]byte, error) {

	analyzer := analysis.NewStatisticalAnalyzer(rg.performanceData)

	descriptiveStats := analyzer.DescriptiveStatistics()

	outliers := analyzer.OutlierDetection()

	report := struct {
		Timestamp time.Time `json:"timestamp"`

		DescriptiveStats analysis.DescriptiveStats `json:"descriptive_stats"`

		Outliers analysis.OutlierResult `json:"outliers"`
	}{

		Timestamp: time.Now(),

		DescriptiveStats: descriptiveStats,

		Outliers: outliers,
	}

	return json.MarshalIndent(report, "", "  ")

}

// GenerateCSVReport exports performance data to CSV.

func (rg *ReportGenerator) GenerateCSVReport(filename string) error {

	file, err := os.Create(filename)

	if err != nil {

		return err

	}

	defer file.Close()

	writer := csv.NewWriter(file)

	defer writer.Flush()

	// Write CSV headers.

	headers := []string{"Timestamp", "Value"}

	if err := writer.Write(headers); err != nil {

		return err

	}

	// Write performance metrics.

	for _, metric := range rg.performanceData {

		record := []string{

			fmt.Sprintf("%d", metric.Timestamp),

			fmt.Sprintf("%f", metric.Value),
		}

		if err := writer.Write(record); err != nil {

			return err

		}

	}

	return nil

}

// GenerateHTMLReport creates a human-readable HTML report.

func (rg *ReportGenerator) GenerateHTMLReport() (string, error) {

	analyzer := analysis.NewStatisticalAnalyzer(rg.performanceData)

	descriptiveStats := analyzer.DescriptiveStatistics()

	outliers := analyzer.OutlierDetection()

	htmlTemplate := `

	<!DOCTYPE html>

	<html>

	<head>

		<title>Performance Analysis Report</title>

		<style>

			body { font-family: Arial, sans-serif; line-height: 1.6; max-width: 800px; margin: 0 auto; padding: 20px; }

			h1 { color: #333; }

			table { width: 100%; border-collapse: collapse; margin-bottom: 20px; }

			th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }

			.pass { color: green; }

			.fail { color: red; }

		</style>

	</head>

	<body>

		<h1>Performance Analysis Report</h1>

		<p>Generated at: {{ .Timestamp }}</p>



		<h2>Descriptive Statistics</h2>

		<table>

			<tr><th>Metric</th><th>Value</th></tr>

			<tr><td>Mean</td><td>{{ .DescriptiveStats.Mean }}</td></tr>

			<tr><td>Median</td><td>{{ .DescriptiveStats.Median }}</td></tr>

			<tr><td>Standard Deviation</td><td>{{ .DescriptiveStats.StdDev }}</td></tr>

			<tr><td>Minimum</td><td>{{ .DescriptiveStats.Min }}</td></tr>

			<tr><td>Maximum</td><td>{{ .DescriptiveStats.Max }}</td></tr>

			<tr><td>95th Percentile</td><td>{{ .DescriptiveStats.Percentile95 }}</td></tr>

		</table>



		<h2>Outlier Analysis</h2>

		<h3>Z-Score Outliers</h3>

		<ul>

			{{ range .Outliers.ZScoreOutliers }}

				<li>{{ . }}</li>

			{{ end }}

		</ul>



		<h3>IQR Outliers</h3>

		<ul>

			{{ range .Outliers.IQROutliers }}

				<li>{{ . }}</li>

			{{ end }}

		</ul>

	</body>

	</html>

	`

	data := struct {
		Timestamp time.Time

		DescriptiveStats analysis.DescriptiveStats

		Outliers analysis.OutlierResult
	}{

		Timestamp: time.Now(),

		DescriptiveStats: descriptiveStats,

		Outliers: outliers,
	}

	tmpl, err := template.New("report").Parse(htmlTemplate)

	if err != nil {

		return "", err

	}

	var renderedReport bytes.Buffer

	if err := tmpl.Execute(&renderedReport, data); err != nil {

		return "", err

	}

	return renderedReport.String(), nil

}
