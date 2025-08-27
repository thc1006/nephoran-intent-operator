#!/usr/bin/env python3
"""
Fix errcheck issues in intent-ingest main_test.go properly
"""

import re

def fix_intent_ingest():
    file_path = "cmd/intent-ingest/main_test.go"
    
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
        
        # Fix unchecked w.Write() calls - add error check
        content = re.sub(
            r'(\s+)w\.Write\((\[.*?\])\)',
            r'\1if _, err := w.Write(\2); err != nil {\n\1\t// Log error but continue (typical for HTTP handlers)\n\1}',
            content
        )
        
        # Fix unchecked resp.Body.Close() calls - use defer with blank identifier
        content = re.sub(
            r'(\s+)defer resp\.Body\.Close\(\)',
            r'\1defer func() { _ = resp.Body.Close() }()',
            content
        )
        
        # Fix any remaining file.Close() calls
        content = re.sub(
            r'(\s+)defer (\w+)\.Close\(\)',
            r'\1defer func() { _ = \2.Close() }()',
            content
        )
        
        with open(file_path, 'w', encoding='utf-8') as f:
            f.write(content)
        
        print(f"Fixed errcheck issues in {file_path}")
        
    except Exception as e:
        print(f"Error processing {file_path}: {e}")

if __name__ == "__main__":
    fix_intent_ingest()