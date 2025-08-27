#!/usr/bin/env python3
"""
Final comprehensive fix for all malformed patterns from bulk errcheck fixes
"""

import os
import re
import glob

def fix_malformed_patterns(file_path):
    """Fix all malformed patterns in a single file"""
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
        
        original_content = content
        
        # Fix patterns like:
        # if test_var
        # = value
        content = re.sub(r'if\s+(\w+)\s*\n\s*=\s+([^;]+)', r'if \1 = \2', content, flags=re.MULTILINE)
        
        # Fix patterns like:
        # _ = // FIXME comment
        # _ = actual_call()
        content = re.sub(r'_ = // FIXME[^\n]*\n\s*_ = ([^\n]+)', r'_ = \1', content, flags=re.MULTILINE)
        
        # Fix patterns like:
        # _ = 
        # actual_call()
        content = re.sub(r'_ = \s*\n\s*([^_\s][^\n]+)', r'_ = \1', content, flags=re.MULTILINE)
        
        # Fix duplicate underscore assignments: _, _ = _, _ = call()
        content = re.sub(r'(_, _ = )(_, _ = )', r'\1', content)
        
        # Fix malformed defer patterns: defer func() { _ = func() { _ = call() }() }()
        content = re.sub(r'defer func\(\) \{ _ = func\(\) \{ _ = ([^}]+) \}\(\) \}\(\)', r'defer func() { _ = \1 }()', content)
        
        # Fix patterns where assignment got split across lines unexpectedly
        # Like: some_var
        #       = value
        content = re.sub(r'(\w+)\s*\n\s*=\s*([^=\n]+(?:\n[^=\n]*)*?)(?=\n\s*\w|\n\s*}|\n\s*$)', r'\1 = \2', content, flags=re.MULTILINE)
        
        # Fix assignment in if statements that got malformed
        content = re.sub(r'if\s+([^=\n]+)\s*\n\s*=\s*([^{;\n]+)\s*{', r'if \1 = \2 {', content, flags=re.MULTILINE)
        
        # Fix function calls that got split
        # pattern: w.Header().Set("Content-Type",
        #          _, _ = "text/plain")
        content = re.sub(r'(\w+\.[\w.()]+\([^)]*,)\s*\n\s*_, _ = ([^)]+\))', r'\1 \2', content, flags=re.MULTILINE)
        
        # Fix cases where FIXME comment broke the line
        content = re.sub(r'([^=\n]+)// FIXME[^\n]*\n\s*([^_\s][^\n]+)', r'\1\2', content, flags=re.MULTILINE)
        
        # Write back if changed
        if content != original_content:
            with open(file_path, 'w', encoding='utf-8') as f:
                f.write(content)
            print(f"Fixed: {file_path}")
            return True
        return False
        
    except Exception as e:
        print(f"Error processing {file_path}: {e}")
        return False

def main():
    # Fix all Go files in cmd/conductor-loop directory
    cmd_dir = "cmd/conductor-loop"
    if os.path.exists(cmd_dir):
        go_files = glob.glob(os.path.join(cmd_dir, "*.go"))
        
        fixed_count = 0
        for file_path in go_files:
            if fix_malformed_patterns(file_path):
                fixed_count += 1
        
        print(f"Fixed {fixed_count} files in {cmd_dir}")
    else:
        print(f"Directory {cmd_dir} not found")

if __name__ == "__main__":
    main()