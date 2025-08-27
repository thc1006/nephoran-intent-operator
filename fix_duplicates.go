package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Fix the duplicate function issues in controllers directory
	controllersDir := "pkg/controllers"
	
	// 1. Remove isConditionTrue duplicate from comprehensive unit test
	comprehensiveFile := filepath.Join(controllersDir, "networkintent_controller_comprehensive_unit_test.go")
	fixFile(comprehensiveFile, map[string]string{
		"func isConditionTrue(conditions []metav1.Condition, conditionType string) bool {\n\tfor _, condition := range conditions {\n\t\tif condition.Type == conditionType {\n\t\t\treturn condition.Status == metav1.ConditionTrue\n\t\t}\n\t}\n\treturn false\n}": "",
		"isConditionTrue(": "testutil.IsConditionTrue(",
	})
	
	// Add testutil import
	addImport(comprehensiveFile, "github.com/thc1006/nephoran-intent-operator/pkg/controllers/testutil")
	
	fmt.Println("Fixed duplicate declarations successfully")
}

func fixFile(filename string, replacements map[string]string) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Printf("Error reading %s: %v", filename, err)
		return
	}
	
	contentStr := string(content)
	for old, new := range replacements {
		contentStr = strings.ReplaceAll(contentStr, old, new)
	}
	
	err = ioutil.WriteFile(filename, []byte(contentStr), 0644)
	if err != nil {
		log.Printf("Error writing %s: %v", filename, err)
	}
}

func addImport(filename string, importPath string) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	
	contentStr := string(content)
	if strings.Contains(contentStr, importPath) {
		return // Already imported
	}
	
	// Simple import addition - add after existing imports
	importPos := strings.Index(contentStr, ")")
	if importPos > 0 {
		before := contentStr[:importPos]
		after := contentStr[importPos:]
		contentStr = before + "\n\t\"" + importPath + "\"" + after
	}
	
	ioutil.WriteFile(filename, []byte(contentStr), 0644)
}