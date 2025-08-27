#!/usr/bin/env python3
"""
Fix all malformed err := // FIXME patterns in porch-direct main_test.go
"""

import os
import re

def fix_porch_direct():
    file_path = "cmd/porch-direct/main_test.go"
    
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
        
        # Fix patterns like:
        # err := // FIXME comment
        # _ = os.WriteFile(...)
        content = re.sub(
            r'err := // FIXME[^\n]*\n\s*_ = (os\.WriteFile[^\n]+)',
            r'err := \1',
            content,
            flags=re.MULTILINE
        )
        
        with open(file_path, 'w', encoding='utf-8') as f:
            f.write(content)
        
        print(f"Fixed malformed patterns in {file_path}")
        
    except Exception as e:
        print(f"Error processing {file_path}: {e}")

if __name__ == "__main__":
    fix_porch_direct()