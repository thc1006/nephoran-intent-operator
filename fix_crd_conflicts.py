#!/usr/bin/env python3
"""
Fix Kubernetes CRD merge conflicts by preserving both HEAD and integrate/mvp changes
"""

def fix_crd_conflicts(file_path):
    with open(file_path, 'r', encoding='utf-8') as f:
        content = f.read()
    
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
            
            # Merge logic: For the specific conflicts we're seeing,
            # we want to keep both maxCPU/maxMemory from HEAD and maxConcurrency from mvp
            # Also preserve the "normal" priority value from HEAD
            
            # Add HEAD content (maxCPU/maxMemory additions)
            for head_line in head_lines:
                if head_line.strip():  # Skip empty lines
                    cleaned_lines.append(head_line)
            
            # Add MVP content (maxConcurrency and other improvements)
            for mvp_line in mvp_lines:
                if mvp_line.strip():  # Skip empty lines
                    # Avoid duplicating priority values
                    if 'medium' not in mvp_line or '- medium' not in head_lines:
                        cleaned_lines.append(mvp_line)
            
        else:
            cleaned_lines.append(line)
            i += 1
    
    # Write the cleaned content back
    with open(file_path, 'w', encoding='utf-8') as f:
        f.write('\n'.join(cleaned_lines))

if __name__ == '__main__':
    fix_crd_conflicts(r'C:\Users\tingy\dev\_worktrees\nephoran\feat-e2e\config\crd\bases\nephoran.com_disasterrecoveryplans.yaml')
    print("Fixed CRD conflicts in disasterrecoveryplans.yaml")