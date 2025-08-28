//go:build windows
// +build windows.

package porch

import (
	"strings"
)

// cmdSafeQuote creates a properly quoted command line for Windows cmd.exe.
//
// When using exec.Command("cmd.exe", "/S", "/C", commandLine), the commandLine.
// must be properly quoted to handle special characters and spaces. Windows cmd.exe.
// has complex quoting rules:.
//
// 1. The /S flag strips outer quotes if the command starts and ends with quotes.
// 2. Special meta-characters (&|()<>^) must be escaped with ^ when not in quotes.
// 3. Inner quotes should be doubled ("") within outer quotes.
// 4. Percent signs must be doubled (%% instead of %).
//
// This function handles these cases to prevent "echo was unexpected at this time" errors.
// and other cmd.exe parsing failures.
//
// References:.
// - https://ss64.com/nt/syntax-esc.html.
// - https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/cmd.
func cmdSafeQuote(args []string) string {
	if len(args) == 0 {
		return ""
	}

	// Build command line directly without excessive escaping.
	var cmdLine string

	// First argument (the command itself).
	if len(args) > 0 {
		cmdLine = quoteIfNeeded(args[0])
	}

	// Add remaining arguments.
	for i := 1; i < len(args); i++ {
		cmdLine += " " + quoteIfNeeded(args[i])
	}

	// If command contains special meta-characters that could be interpreted by cmd,.
	// wrap the entire command in quotes.
	if needsOuterWrapping(cmdLine) {
		// For /S /C, we need to wrap in quotes and double any inner quotes.
		cmdLine = strings.ReplaceAll(cmdLine, `"`, `""`)
		cmdLine = `"` + cmdLine + `"`
	}

	return cmdLine
}

// quoteWindowsArg quotes a single argument for Windows cmd.exe.
func quoteWindowsArg(arg string) string {
	// Empty argument needs quotes.
	if arg == "" {
		return `""`
	}

	// Check if argument needs quoting.
	needsQuotes := false
	hasMetaChars := false

	for _, r := range arg {
		switch r {
		case ' ', '\t', '\n', '\v', '"':
			needsQuotes = true
		case '&', '|', '(', ')', '<', '>', '^':
			hasMetaChars = true
			needsQuotes = true
		case '%':
			// Percent needs special handling.
			hasMetaChars = true
		}
	}

	// If no special characters, return as-is.
	if !needsQuotes && !hasMetaChars {
		return arg
	}

	// Escape meta-characters.
	if hasMetaChars {
		arg = escapeMetaChars(arg)
	}

	// Add quotes if needed.
	if needsQuotes {
		// Escape any quotes in the argument.
		arg = strings.ReplaceAll(arg, `"`, `""`)
		arg = `"` + arg + `"`
	}

	return arg
}

// escapeMetaChars escapes cmd.exe meta-characters with ^.
func escapeMetaChars(s string) string {
	var result strings.Builder
	for _, r := range s {
		switch r {
		case '&', '|', '(', ')', '<', '>', '^':
			result.WriteRune('^')
			result.WriteRune(r)
		case '%':
			// Double percent signs for cmd.exe.
			result.WriteString("%%")
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}

// quoteIfNeeded quotes an argument only if it needs quoting.
func quoteIfNeeded(arg string) string {
	if arg == "" {
		return `""`
	}

	// Check if argument needs quoting.
	needsQuotes := false
	for _, r := range arg {
		switch r {
		case ' ', '\t', '&', '|', '(', ')', '<', '>', '^', '"':
			needsQuotes = true
			break
		}
	}

	// Handle percent signs - always double them.
	if strings.Contains(arg, "%") {
		arg = strings.ReplaceAll(arg, "%", "%%")
	}

	if !needsQuotes {
		return arg
	}

	// Escape special characters within the argument.
	var result strings.Builder
	for _, r := range arg {
		switch r {
		case '&', '|', '(', ')', '<', '>', '^':
			result.WriteRune('^')
			result.WriteRune(r)
		case '"':
			result.WriteString(`""`)
		default:
			result.WriteRune(r)
		}
	}

	return `"` + result.String() + `"`
}

// needsOuterWrapping checks if the entire command needs to be wrapped in quotes.
func needsOuterWrapping(cmdLine string) bool {
	// Check for unquoted special characters.
	inQuotes := false
	for _, r := range cmdLine {
		if r == '"' {
			inQuotes = !inQuotes
			continue
		}
		if !inQuotes {
			switch r {
			case '&', '|', '(', ')', '<', '>':
				return true
			}
		}
	}
	return false
}
