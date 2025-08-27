#!/usr/bin/env python3
"""
Fix final malformed patterns from bulk errcheck fixes
"""

import os
import re
import glob

def fix_final_patterns(file_path):
    """Fix final malformed patterns in a single file"""
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
        
        original_content = content
        
        # Fix patterns like:
        # require.NoError(t, // FIXME comment
        # _ = os.WriteFile(...))
        content = re.sub(
            r'(require\.NoError\(t,)\s*// FIXME[^\n]*\n\s*_ = ([^)]+\))',
            r'\1 \2',
            content,
            flags=re.MULTILINE
        )
        
        # Fix patterns like:
        # assert.NoError(t, // FIXME comment
        # _ = someCall())
        content = re.sub(
            r'(assert\.NoError\(t,)\s*// FIXME[^\n]*\n\s*_ = ([^)]+\))',
            r'\1 \2',
            content,
            flags=re.MULTILINE
        )
        
        # Fix patterns like:
        # someFunc(arg1, // FIXME comment
        # _ = arg2)
        content = re.sub(
            r'(\w+\([^)]*,)\s*// FIXME[^\n]*\n\s*_ = ([^)]+\))',
            r'\1 \2',
            content,
            flags=re.MULTILINE
        )
        
        # Fix remaining _ = statements that should just be statements
        # Pattern: _ = someCall() should become someCall() where appropriate
        lines = content.split('\n')
        fixed_lines = []
        
        for i, line in enumerate(lines):
            # If line is just "_ = someCall()" and is standalone, make it just "someCall()"
            stripped = line.strip()
            if (stripped.startswith('_ = ') and 
                not stripped.startswith('_ = _, ') and 
                '(' in stripped and ')' in stripped and
                stripped.endswith(')')):
                # Get indentation
                indent = line[:len(line) - len(line.lstrip())]
                call = stripped[4:]  # Remove "_ = "
                fixed_lines.append(indent + call)
            else:
                fixed_lines.append(line)
        
        content = '\n'.join(fixed_lines)
        
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
            if fix_final_patterns(file_path):
                fixed_count += 1
        
        print(f"Fixed {fixed_count} files in {cmd_dir}")
    else:
        print(f"Directory {cmd_dir} not found")

if __name__ == "__main__":
    main()