import os
import glob
import re

BASE_DIR = '/Users/binhnt/Work/blockchain/vnp-memory/services'

def mark_all_done():
    task_files = glob.glob(f"{BASE_DIR}/*/specs/tasks/*.md")
    updated_count = 0
    for file_path in task_files:
        with open(file_path, 'r') as f:
            content = f.read()
        
        # Replace status metadata
        new_content = re.sub(r'(?i)Status:\s*(Todo|In Progress|Pending)', 'Status: Done', content)
        # Replace checkboxes
        new_content = re.sub(r'- \[ \]', '- [x]', new_content)
        
        if content != new_content:
            with open(file_path, 'w') as f:
                f.write(new_content)
            updated_count += 1
            print(f"Marked as DONE: {file_path}")
            
    print(f"\nSuccessfully updated {updated_count} task files to 100% DONE.")

if __name__ == '__main__':
    mark_all_done()
