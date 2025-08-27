#!/usr/bin/env python3
"""
Fix malformed defer func patterns created by the bulk errcheck fix.
"""

import os
import re
import glob

def fix_malformed_defer_patterns(filepath):
    """Fix malformed defer patterns in a file."""
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()
    
    original_content = content
    
    # Fix patterns like: defer func() { _ = func() { _ = someCall() }() }()
    # Replace with: defer func() { _ = someCall() }()
    pattern = r'defer func\(\) \{ _ = func\(\) \{ _ = ([^}]+)\(\) \}\(\) \}\(\)'
    replacement = r'defer func() { _ = \1() }()'
    content = re.sub(pattern, replacement, content)
    
    # Fix patterns in the graceful_shutdown_test.go that are incomplete
    pattern2 = r'defer func\(\) \{ _ = func\(\) \{ _ = // FIXME: Adding error check per errcheck linter\n _ = ([^}]+)\(\) \}\(\) \}\(\)'
    replacement2 = r'defer func() { _ = \1() }()'
    content = re.sub(pattern2, replacement2, content, flags=re.MULTILINE)
    
    if content != original_content:
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(content)
        print(f"Fixed: {filepath}")
        return True
    return False

def main():
    """Fix all malformed patterns in cmd directory."""
    fixed_count = 0
    
    for filepath in glob.glob("cmd/**/*.go", recursive=True):
        if fix_malformed_defer_patterns(filepath):
            fixed_count += 1
    
    print(f"Fixed {fixed_count} files")

if __name__ == "__main__":
    main()