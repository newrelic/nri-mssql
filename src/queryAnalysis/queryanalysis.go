package queryAnalysis

import "fmt"

// RunAnalysis runs all types of analyses
func RunAnalysis() {
	fmt.Println("Starting query analysis...")

	AnalyzeSlowQueries()
	AnalyzeWaits()
	AnalyzeExecutionPlans()

	fmt.Println("Query analysis completed.")
}
