import os
import shutil
import subprocess

services_dir = "/Users/binhnt/Work/blockchain/vnp-memory/services"

targets = [
    "vnp-search-hub", "sm-memory", "graphiti-store", "cognee-search", "memobase-context", "vnp-event", 
    "memobase-ingestion", "sm-connector", "sm-engine", "zep-core", "ov-session", 
    "vnp-admin", "vnp-pipelines", "vnp-infra", "vnp-observability"
]

def refactor_service(service_path):
    internal_dir = os.path.join(service_path, "internal")
    if not os.path.isdir(internal_dir):
        return

    print(f"Refactoring {service_path}...")
    
    mod_name = ""
    go_mod_path = os.path.join(service_path, "go.mod")
    if os.path.exists(go_mod_path):
        with open(go_mod_path, "r") as f:
            for line in f:
                if line.startswith("module "):
                    mod_name = line.strip().split(" ")[1]
                    break
    
    if not mod_name:
        return

    for item in os.listdir(internal_dir):
        s = os.path.join(internal_dir, item)
        d = os.path.join(service_path, item)
        if os.path.exists(d):
            if os.path.isdir(s) and os.path.isdir(d):
                for sub_item in os.listdir(s):
                    shutil.move(os.path.join(s, sub_item), os.path.join(d, sub_item))
            else:
                shutil.move(s, d)
        else:
            shutil.move(s, d)
    
    try:
        os.rmdir(internal_dir)
    except OSError:
        print(f"Could not remove {internal_dir}")

    old_import = f"{mod_name}/internal"
    new_import = f"{mod_name}"
    
    for root, dirs, files in os.walk(service_path):
        for file in files:
            if file.endswith(".go"):
                file_path = os.path.join(root, file)
                with open(file_path, "r") as f:
                    content = f.read()
                
                if old_import in content:
                    content = content.replace(old_import, new_import)
                    with open(file_path, "w") as f:
                        f.write(content)

    env = os.environ.copy()
    env["PATH"] = "/opt/homebrew/bin:/usr/local/go/bin:" + env.get("PATH", "")
    subprocess.run(["go", "mod", "tidy"], cwd=service_path, env=env)

for t in targets:
    service_path = os.path.join(services_dir, t)
    if os.path.isdir(service_path):
        refactor_service(service_path)

print("Done.")
