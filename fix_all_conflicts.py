#!/usr/bin/env python3
"""
Fix all merge conflicts by preserving both HEAD and integrate/mvp changes intelligently
"""
import os
import re
import subprocess
import sys

def find_conflicted_files(root_dir):
    """Find all files with merge conflict markers"""
    conflicted_files = []
    
    # Run find command to get all files with conflict markers
    try:
        result = subprocess.run([
            'find', root_dir, '-name', '*.go', '-exec', 
            'grep', '-l', '<<<<<<< HEAD\\|=======\\|>>>>>>> integrate/mvp', '{}', ';'
        ], capture_output=True, text=True)
        
        if result.stdout:
            conflicted_files = result.stdout.strip().split('\n')
            
    except Exception as e:
        print(f"Error finding conflicted files: {e}")
        # Fallback: manually check some files
        return []
    
    return [f for f in conflicted_files if f.strip()]

def fix_go_file_conflicts(file_path):
    """Fix conflicts in Go files by intelligently merging changes"""
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
    except Exception as e:
        print(f"Error reading {file_path}: {e}")
        return False
    
    lines = content.split('\n')
    cleaned_lines = []
    i = 0
    
    while i < len(lines):
        line = lines[i]
        
        if line.strip() == '<<<<<<< HEAD':
            # Start of conflict - collect HEAD section
            i += 1
            head_lines = []
            while i < len(lines) and lines[i].strip() != '=======':
                head_lines.append(lines[i])
                i += 1
            
            # Skip the ======= marker
            i += 1
            
            # Collect integrate/mvp section
            mvp_lines = []
            while i < len(lines) and lines[i].strip() != '>>>>>>> integrate/mvp':
                mvp_lines.append(lines[i])
                i += 1
            
            # Skip the >>>>>>> marker
            i += 1
            
            # Merge logic for different scenarios
            merged_section = merge_sections(head_lines, mvp_lines, file_path)
            cleaned_lines.extend(merged_section)
            
        else:
            cleaned_lines.append(line)
            i += 1
    
    # Write the cleaned content back
    try:
        with open(file_path, 'w', encoding='utf-8') as f:
            f.write('\n'.join(cleaned_lines))
        return True
    except Exception as e:
        print(f"Error writing {file_path}: {e}")
        return False

def merge_sections(head_lines, mvp_lines, file_path):
    """Intelligently merge HEAD and MVP sections"""
    merged = []
    
    # Remove empty lines from both sections for comparison
    head_non_empty = [line for line in head_lines if line.strip()]
    mvp_non_empty = [line for line in mvp_lines if line.strip()]
    
    # For import statements, merge both
    if any('import' in line for line in head_lines + mvp_lines):
        # Collect all imports
        head_imports = set()
        mvp_imports = set()
        other_lines = []
        
        for line in head_lines:
            if line.strip().startswith('"') and line.strip().endswith('"'):
                head_imports.add(line.strip())
            else:
                other_lines.append(line)
        
        for line in mvp_lines:
            if line.strip().startswith('"') and line.strip().endswith('"'):
                mvp_imports.add(line.strip())
            else:
                if line not in other_lines:
                    other_lines.append(line)
        
        # Add all imports, sorted
        all_imports = sorted(head_imports | mvp_imports)
        for imp in all_imports:
            merged.append(f'\t{imp}')
        
        # Add other lines
        merged.extend(other_lines)
        
    else:
        # For other conflicts, prefer HEAD but include meaningful MVP additions
        merged.extend(head_lines)
        
        # Add MVP lines that aren't duplicates or empty
        for mvp_line in mvp_lines:
            if mvp_line.strip() and mvp_line not in head_lines:
                # Check if it's a meaningful addition (not just whitespace changes)
                if any(meaningful_content in mvp_line.lower() for meaningful_content in 
                       ['func', 'var', 'const', 'type', 'return', 'if', 'for', 'switch']):
                    merged.append(mvp_line)
    
    return merged

def main():
    root_dir = r'C:\Users\tingy\dev\_worktrees\nephoran\feat-e2e'
    
    # Find all conflicted files
    conflicted_files = find_conflicted_files(root_dir)
    
    if not conflicted_files:
        print("No conflicted files found!")
        return
    
    print(f"Found {len(conflicted_files)} files with conflicts:")
    for f in conflicted_files[:10]:  # Show first 10
        print(f"  {f}")
    if len(conflicted_files) > 10:
        print(f"  ... and {len(conflicted_files) - 10} more")
    
    print("\nFixing conflicts...")
    fixed_count = 0
    
    for file_path in conflicted_files:
        if fix_go_file_conflicts(file_path):
            fixed_count += 1
            print(f"✓ Fixed {os.path.basename(file_path)}")
        else:
            print(f"✗ Failed to fix {os.path.basename(file_path)}")
    
    print(f"\nFixed {fixed_count}/{len(conflicted_files)} files")

if __name__ == '__main__':
    main()