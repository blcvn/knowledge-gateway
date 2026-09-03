# Python Scripts Rule

All supporting Python scripts (such as refactoring scripts, codebase manipulation scripts, temporary migration tools, etc.) MUST be placed inside the `scripts/` directory.

## Rules
1. **No Root Scripts**: Do NOT place any new `.py` script files in the project root directory (`/Users/binhnt/Work/blockchain/vnp-memory/`).
2. **Directory**: Always use the `/Users/binhnt/Work/blockchain/vnp-memory/scripts/` directory to store `.py` scripts. 
3. **Execution**: If you need to write and execute a script to aid your tasks, create it inside `scripts/`, execute it from the root or from `scripts/`, and keep it there if it might be useful again, or delete it after use if it's purely temporary.
