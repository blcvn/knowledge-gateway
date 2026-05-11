import os

SERVICES_TO_DEPRECATE_ENGINE = ['sm-document', 'sm-memory', 'sm-profile']
SERVICE_TO_DEPRECATE_GATEWAY = ['sm-mcp']

def update_deprecated(service_name, merge_target, arch_spec):
    readme_path = f"services/{service_name}/docs/README.md"
    arch_path = f"services/{service_name}/docs/architecture.md"
    tdd_path = f"services/{service_name}/specs/tdd.md"

    # Update README
    if os.path.exists(readme_path):
        with open(readme_path, 'r') as f:
            content = f.read()
        content = content.replace("status: Active", "status: Deprecated")
        warning = f"\n> **🚨 DEPRECATION NOTICE**: This service has been merged into `{merge_target}`. See [{arch_spec}](../../../specs/architecture/{arch_spec}.md) for details.\n\n"
        content = content.replace("## Purpose\n", f"## Purpose\n{warning}")
        with open(readme_path, 'w') as f:
            f.write(content)

    # Update Architecture
    if os.path.exists(arch_path):
        with open(arch_path, 'r') as f:
            content = f.read()
        content = content.replace("status: Active", "status: Deprecated")
        warning = f"\n> **🚨 DEPRECATION NOTICE**: This architecture is obsolete. The service has been merged into `{merge_target}` (Ref: [{arch_spec}]).\n\n"
        if "# " in content:
            content = content.replace("\n## ", f"{warning}\n## ", 1)
        with open(arch_path, 'w') as f:
            f.write(content)

    # Update TDD
    if os.path.exists(tdd_path):
        with open(tdd_path, 'r') as f:
            content = f.read()
        content = content.replace("status: Draft", "status: Deprecated").replace("status: Active", "status: Deprecated")
        warning = f"\n> **🚨 DEPRECATION NOTICE**: This specification is obsolete. The service has been merged into `{merge_target}` (Ref: [{arch_spec}]).\n\n"
        if "# " in content:
            content = content.replace("\n## ", f"{warning}\n## ", 1)
        with open(tdd_path, 'w') as f:
            f.write(content)

for s in SERVICES_TO_DEPRECATE_ENGINE:
    update_deprecated(s, 'sm-engine', 'ARCH-007-merge-sm-engine')

for s in SERVICE_TO_DEPRECATE_GATEWAY:
    update_deprecated(s, 'vnp-gateway', 'ARCH-008-absorb-sm-mcp-to-gateway')

print("Deprecated services updated.")
