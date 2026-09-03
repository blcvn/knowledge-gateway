import os
import sys

replacements = {
    "vnp-memory/pkg/": "vnp-memory/shared/pkg/",
    "github.com/vnp-community/vnp-memory/pkg/": "github.com/vnp-community/vnp-memory/shared/pkg/",
    "./pkg/": "./shared/pkg/" # specifically for go.work
}

def process_file(filepath):
    try:
        with open(filepath, 'r') as f:
            content = f.read()
            
        new_content = content
        for old, new in replacements.items():
            new_content = new_content.replace(old, new)
            
        if new_content != content:
            with open(filepath, 'w') as f:
                f.write(new_content)
            print(f"Updated {filepath}")
            return True
        return False
    except Exception as e:
        print(f"Error processing {filepath}: {e}")
        return False

updated_count = 0
for root, dirs, files in os.walk('.'):
    # skip hidden dirs and binaries
    if '.git' in root or '.antigravity' in root or '.claude' in root or '.idea' in root:
        continue
    for file in files:
        if file.endswith('.go') or file.endswith('.mod') or file.endswith('.work') or file.endswith('.sh') or file.endswith('.yaml'):
            filepath = os.path.join(root, file)
            if process_file(filepath):
                updated_count += 1

print(f"Finished updating {updated_count} files.")
