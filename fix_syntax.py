#!/usr/bin/env python3
"""
Fix syntax errors from the bulk errcheck replacement script.
"""

import os
import re
import glob

def fix_syntax_errors(filepath):
    """Fix syntax errors in a Go file."""
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()
    
    original_content = content
    
    # Fix malformed if err patterns like: if err := // FIXME: ...\n _ = someCall(); err != nil {
    pattern1 = r'if err := // FIXME: Adding error check per errcheck linter\n _ = ([^;]+);(.+)'
    replacement1 = r'if err := \1;\2'
    content = re.sub(pattern1, replacement1, content, flags=re.MULTILINE)
    
    # Fix malformed assignments like: _, _ = _, _ = someCall()
    pattern2 = r'_, _ = _, _ = ([^)]+\([^)]*\))'
    replacement2 = r'_, _ = \1'
    content = re.sub(pattern2, replacement2, content)
    
    # Fix malformed function calls with assignments: someFunc("param", _, _ = "value")
    pattern3 = r'([^(]+)\("([^"]+)", // FIXME: Adding error check per errcheck linter\n _, _ = "([^"]+)"\)'
    replacement3 = r'\1("\2", "\3")'
    content = re.sub(pattern3, replacement3, content, flags=re.MULTILINE)
    
    # Fix more complex malformed patterns
    pattern4 = r'([^(]+)\(([^,]+), // FIXME: Adding error check per errcheck linter\n _, _ = ([^)]+)\)'
    replacement4 = r'\1(\2, \3)'
    content = re.sub(pattern4, replacement4, content, flags=re.MULTILINE)
    
    if content != original_content:
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(content)
        print(f"Fixed syntax: {filepath}")
        return True
    return False

def main():
    """Fix all syntax errors in cmd directory."""
    fixed_count = 0
    
    for filepath in glob.glob("cmd/**/*.go", recursive=True):
        if fix_syntax_errors(filepath):
            fixed_count += 1
    
    print(f"Fixed syntax in {fixed_count} files")

if __name__ == "__main__":
    main()