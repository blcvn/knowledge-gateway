import os

SERVICES = [
    'sm-analytics',
    'sm-auth',
    'sm-connector',
    'sm-document',
    'sm-engine',
    'sm-mcp',
    'sm-memory',
    'sm-profile',
    'sm-project',
    'sm-search'
]

BASE_DIR = '/Users/binhnt/Work/blockchain/vnp-memory/services'

GOMOD_TEMPLATE = """module vnp-memory/services/{service}

go 1.23.0

require (
\tgoogle.golang.org/grpc v1.65.0
\tgoogle.golang.org/protobuf v1.34.2
)
"""

for svc in SERVICES:
    svc_dir = os.path.join(BASE_DIR, svc)
    go_mod_path = os.path.join(svc_dir, 'go.mod')
    
    if not os.path.exists(go_mod_path):
        print(f"Creating go.mod for {svc}...")
        with open(go_mod_path, 'w') as f:
            f.write(GOMOD_TEMPLATE.format(service=svc))
            
    print(f"Initialized {svc} as a Go module.")
