#!/usr/bin/env bash
###############################################################################
# VNP Memory — Full Deployment Script for Dev Server (172.20.2.39)
#
# LOCAL COMPILE + BINARY SYNC deployment model:
# - All Go apps are cross-compiled locally for linux/amd64
# - Frontend (UI) is built locally with npm
# - Compiled binaries + configs are synced to server via rsync
# - Containers mount binaries from host filesystem (no Docker build needed)
#
# Usage:
#   ./deploy.sh                   # Full deploy (all services)
#   ./deploy.sh --monolith        # Deploy Memory Monolith only (API+UI)
#   ./deploy.sh --monolith-full   # Deploy Monolith + sync nginx on gateway
#   ./deploy.sh --kgs             # Deploy KGS Platform only
#   ./deploy.sh --config-only     # Sync config & restart (no recompile)
#   ./deploy.sh --compile-only    # Compile only (no sync/restart)
#   ./deploy.sh --sync-only       # Sync only (no compile/restart)
#   ./deploy.sh --nginx           # Sync nginx config to gateway server
#   ./deploy.sh --setup           # First-time setup (build runtime image)
###############################################################################
set -euo pipefail

# ── Configuration ────────────────────────────────────────────────────────────
DEV_SERVER="172.20.2.39"
DEV_USER="ubuntu"
DEPLOY_DIR="/opt/vnp-memory"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MONOREPO_ROOT="${SCRIPT_DIR}/../.."

# Output directories (local)
BIN_DIR="${SCRIPT_DIR}/bin"
UI_DIST_DIR="${SCRIPT_DIR}/ui-dist"

# Gateway server (nginx + certbot)
GATEWAY_SERVER="172.20.2.16"
GATEWAY_USER="ubuntu"

# Cross-compile flags
export CGO_ENABLED=0
export GOOS=linux
export GOARCH=amd64
GO_LDFLAGS="-ldflags=-s -w"

# App source directories (relative to monorepo root)
MEMORY_APP="apps/memory"
COGNEE_APP="apps/cognee"
GRAPHITI_APP="apps/graphiti"
OPENVIKING_APP="apps/OpenViking"
ZEP_APP="apps/zep"
MEMOBASE_APP="apps/memobase"
SUPERMEMORY_APP="apps/supermemory"
KGS_APP="kgs-platform"

# ── Colors ───────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

log()  { echo -e "${BLUE}[INFO]${NC}  $*"; }
ok()   { echo -e "${GREEN}[OK]${NC}    $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC}  $*"; }
err()  { echo -e "${RED}[ERROR]${NC} $*" >&2; }
step() { echo -e "${CYAN}  →${NC} $*"; }

# ── Compile Functions ────────────────────────────────────────────────────────

compile_app() {
    local name="$1"
    local app_dir="$2"
    local build_target="$3"
    local output_name="$4"

    log "Compiling ${name} (linux/amd64) ..."
    mkdir -p "${BIN_DIR}"

    cd "${MONOREPO_ROOT}/${app_dir}"
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
        -ldflags="-s -w" \
        -o "${BIN_DIR}/${output_name}" \
        "${build_target}"

    local size
    size=$(du -sh "${BIN_DIR}/${output_name}" | awk '{print $1}')
    ok "${name} compiled → bin/${output_name} (${size})"
}

compile_kgs() {
    log "Compiling KGS Platform (linux/amd64) ..."
    mkdir -p "${BIN_DIR}"

    cd "${MONOREPO_ROOT}/${KGS_APP}"
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
        -mod=mod \
        -ldflags="-s -w" \
        -o "${BIN_DIR}/kgs-server" \
        ./cmd/server/

    local size
    size=$(du -sh "${BIN_DIR}/kgs-server" | awk '{print $1}')
    ok "KGS Platform compiled → bin/kgs-server (${size})"
}

compile_ui() {
    log "Building UI frontend ..."
    cd "${MONOREPO_ROOT}/ui"
    npm install
    npm run build
    ok "UI built: ui/dist/"
}

compile_memory_monolith() {
    # Step 1: Build UI
    compile_ui

    # Step 2: Embed UI assets into Go binary path
    log "Embedding UI assets into Memory Monolith ..."
    rm -rf "${MONOREPO_ROOT}/${MEMORY_APP}/internal/ui/ui_dist"
    mkdir -p "${MONOREPO_ROOT}/${MEMORY_APP}/internal/ui/ui_dist"
    cp -r "${MONOREPO_ROOT}/ui/dist/"* "${MONOREPO_ROOT}/${MEMORY_APP}/internal/ui/ui_dist/"
    ok "UI assets embedded"

    # Step 3: Compile Memory Monolith from monorepo root (needs go.work)
    log "Compiling Memory Monolith (linux/amd64) ..."
    mkdir -p "${BIN_DIR}"

    cd "${MONOREPO_ROOT}"
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
        -ldflags="-s -w" \
        -o "${BIN_DIR}/vnp-memory" \
        "./${MEMORY_APP}/cmd/server"

    local size
    size=$(du -sh "${BIN_DIR}/vnp-memory" | awk '{print $1}')
    ok "Memory Monolith compiled → bin/vnp-memory (${size})"
}

compile_all_apps() {
    compile_app "Cognee"      "${COGNEE_APP}"      "./cmd/cognee/"     "cognee"
    compile_app "Graphiti"    "${GRAPHITI_APP}"     "./cmd/graphiti/"   "graphiti"
    compile_app "OpenViking"  "${OPENVIKING_APP}"   "./cmd/openviking/" "openviking"
    compile_app "Zep"         "${ZEP_APP}"          "./cmd/zep/"        "zep"
    compile_app "Memobase"    "${MEMOBASE_APP}"     "./cmd/memobase/"   "memobase"
    compile_app "Supermemory" "${SUPERMEMORY_APP}"  "./cmd/supermemory/" "supermemory"
}

compile_all() {
    compile_all_apps
    compile_kgs
    compile_memory_monolith

    echo ""
    ok "All services compiled!"
    echo ""
    ls -lh "${BIN_DIR}/"
    echo ""
}

# ── Sync Functions ───────────────────────────────────────────────────────────

sync_binaries() {
    log "Syncing binaries to ${DEV_SERVER}:${DEPLOY_DIR}/bin/ ..."
    ssh "${DEV_USER}@${DEV_SERVER}" "mkdir -p ${DEPLOY_DIR}/bin"
    rsync -avz --progress "${BIN_DIR}/" "${DEV_USER}@${DEV_SERVER}:${DEPLOY_DIR}/bin/"
    ssh "${DEV_USER}@${DEV_SERVER}" "chmod +x ${DEPLOY_DIR}/bin/*"
    ok "Binaries synced to ${DEV_SERVER}"
}

sync_ui() {
    log "Syncing UI dist to ${DEV_SERVER}:${DEPLOY_DIR}/ui-dist/ ..."
    ssh "${DEV_USER}@${DEV_SERVER}" "mkdir -p ${DEPLOY_DIR}/ui-dist"
    rsync -avz --delete --progress "${MONOREPO_ROOT}/ui/dist/" "${DEV_USER}@${DEV_SERVER}:${DEPLOY_DIR}/ui-dist/"
    ok "UI dist synced to ${DEV_SERVER}"
}

sync_config() {
    log "Syncing configuration to ${DEV_SERVER}:${DEPLOY_DIR} ..."
    ssh "${DEV_USER}@${DEV_SERVER}" "mkdir -p ${DEPLOY_DIR}/config"

    scp "${SCRIPT_DIR}/docker-compose.yml" "${DEV_USER}@${DEV_SERVER}:${DEPLOY_DIR}/"
    scp "${SCRIPT_DIR}/Dockerfile.runtime" "${DEV_USER}@${DEV_SERVER}:${DEPLOY_DIR}/"
    scp "${SCRIPT_DIR}/init-db.sql"        "${DEV_USER}@${DEV_SERVER}:${DEPLOY_DIR}/"
    scp "${SCRIPT_DIR}/config/zep.yaml"    "${DEV_USER}@${DEV_SERVER}:${DEPLOY_DIR}/config/"
    scp "${SCRIPT_DIR}/config/kgs.yaml"    "${DEV_USER}@${DEV_SERVER}:${DEPLOY_DIR}/config/"
    scp "${SCRIPT_DIR}/config/memory.yaml" "${DEV_USER}@${DEV_SERVER}:${DEPLOY_DIR}/config/"

    if [ -f "${SCRIPT_DIR}/.env" ]; then
        scp "${SCRIPT_DIR}/.env" "${DEV_USER}@${DEV_SERVER}:${DEPLOY_DIR}/"
        ok "Synced .env + configs"
    else
        scp "${SCRIPT_DIR}/.env.example" "${DEV_USER}@${DEV_SERVER}:${DEPLOY_DIR}/.env"
        warn "No .env found — synced .env.example as .env"
    fi
}

sync_nginx() {
    log "Syncing nginx config to ${GATEWAY_SERVER} ..."
    scp "${SCRIPT_DIR}/nginx/c6-openledger-vn.conf" "${GATEWAY_USER}@${GATEWAY_SERVER}:/tmp/c6-openledger-vn.conf"
    ssh "${GATEWAY_USER}@${GATEWAY_SERVER}" "\
        NGINX_CONTAINER=\$(docker ps -qf \"name=nginx\" | head -n 1) && \
        if [ -z \"\$NGINX_CONTAINER\" ]; then echo 'Nginx container not found'; exit 1; fi && \
        mv /tmp/c6-openledger-vn.conf /home/ubuntu/vnp-qa-platform/proxy/conf.d/c6-openledger-vn.conf && \
        rm -f /home/ubuntu/vnp-qa-platform/proxy/conf.d/c6-openledger-nginx.conf && \
        docker exec \$NGINX_CONTAINER nginx -t && \
        docker exec \$NGINX_CONTAINER nginx -s reload"
    ok "Nginx config updated and reloaded on ${GATEWAY_SERVER}"
    echo "  https://c6.openledger.vn → ${DEV_SERVER}:8080"
}

sync_all() {
    sync_binaries
    sync_config
}

# ── Remote Functions ─────────────────────────────────────────────────────────

remote_stop() {
    log "Stopping services on ${DEV_SERVER} ..."
    ssh "${DEV_USER}@${DEV_SERVER}" \
        "cd ${DEPLOY_DIR} && docker compose --profile monolith --profile memory --profile kgs --profile surrealdb --profile ui --profile monitoring down" \
        || true
}

remote_stop_monolith() {
    log "Stopping Memory Monolith on ${DEV_SERVER} ..."
    ssh "${DEV_USER}@${DEV_SERVER}" \
        "cd ${DEPLOY_DIR} && docker compose --profile monolith down" \
        || true
}

remote_stop_kgs() {
    log "Stopping KGS Platform on ${DEV_SERVER} ..."
    ssh "${DEV_USER}@${DEV_SERVER}" \
        "cd ${DEPLOY_DIR} && docker compose --profile kgs down" \
        || true
}

remote_start() {
    log "Starting services on ${DEV_SERVER} ..."
    ssh "${DEV_USER}@${DEV_SERVER}" \
        "cd ${DEPLOY_DIR} && docker compose --profile memory --profile ui up -d --pull never"
    ok "Services started!"
}

remote_start_monolith() {
    log "Starting Memory Monolith on ${DEV_SERVER} ..."
    ssh "${DEV_USER}@${DEV_SERVER}" \
        "cd ${DEPLOY_DIR} && docker compose --profile monolith up -d --pull never"
    ok "Memory Monolith started!"
}

remote_start_kgs() {
    log "Starting KGS Platform on ${DEV_SERVER} ..."
    ssh "${DEV_USER}@${DEV_SERVER}" \
        "cd ${DEPLOY_DIR} && docker compose --profile kgs up -d --pull never"
    ok "KGS Platform started!"
}

remote_pull() {
    log "Pulling latest public images on ${DEV_SERVER} ..."
    ssh "${DEV_USER}@${DEV_SERVER}" \
        "cd ${DEPLOY_DIR} && docker compose pull neo4j postgres redis qdrant" \
        || true
}

setup_runtime() {
    log "Building runtime base image on ${DEV_SERVER} ..."
    sync_config
    ssh "${DEV_USER}@${DEV_SERVER}" \
        "cd ${DEPLOY_DIR} && docker build -t vnp/runtime:latest -f Dockerfile.runtime ."
    ok "vnp/runtime:latest built on ${DEV_SERVER}"
}

# ── Main ─────────────────────────────────────────────────────────────────────

MODE="${1:-full}"

case "${MODE}" in
    --setup)
        log "🔧 First-time setup: building runtime base image"
        echo ""
        setup_runtime
        echo ""
        ok "🎉 Setup complete! Runtime image ready on ${DEV_SERVER}"
        echo ""
        echo "  Next steps:"
        echo "  1. ./deploy.sh --monolith    # Deploy Memory Monolith"
        echo "  2. ./deploy.sh               # Deploy all services"
        echo ""
        ;;
    --monolith)
        log "🧊 Memory Monolith deployment"
        echo ""
        compile_memory_monolith
        sync_binaries
        sync_config
        remote_stop_monolith
        remote_start_monolith
        echo ""
        ok "🎉 Memory Monolith deployed!"
        echo ""
        echo "  REST API + UI: http://${DEV_SERVER}:8080"
        echo "  MCP:           http://${DEV_SERVER}:8082"
        echo "  Health:        http://${DEV_SERVER}:8083/healthz"
        echo ""
        echo "  💡 Run './deploy.sh --nginx' to update nginx on gateway server."
        echo ""
        ;;
    --monolith-full)
        log "🧊 Memory Monolith + Nginx full deployment"
        echo ""
        compile_memory_monolith
        sync_binaries
        sync_config
        remote_stop_monolith
        remote_start_monolith
        sync_nginx
        echo ""
        ok "🎉 Full Monolith deployment complete!"
        echo ""
        echo "  Public URL:    https://c6.openledger.vn"
        echo "  REST API:      https://c6.openledger.vn/v1/"
        echo "  UI Console:    https://c6.openledger.vn/"
        echo "  MCP:           https://c6.openledger.vn/mcp/"
        echo ""
        ;;
    --kgs)
        log "📊 KGS Platform deployment"
        echo ""
        compile_kgs
        sync_binaries
        sync_config
        remote_stop_kgs
        remote_start_kgs
        echo ""
        ok "🎉 KGS Platform deployed!"
        echo ""
        echo "  HTTP: http://${DEV_SERVER}:8010"
        echo "  gRPC: http://${DEV_SERVER}:9010"
        echo ""
        ;;
    --nginx)
        log "🌐 Nginx config sync only"
        sync_nginx
        ;;
    --config-only)
        log "🔧 Config-only deployment"
        sync_config
        remote_stop
        remote_start
        ;;
    --compile-only)
        log "📦 Compile-only (no sync/restart)"
        compile_all
        ;;
    --sync-only)
        log "🚀 Sync-only (no compile/restart)"
        sync_all
        ;;
    --pull)
        log "⬇️  Pull latest public images"
        remote_pull
        ;;
    full|*)
        log "🚀 Full deployment to ${DEV_SERVER}"
        echo ""

        # Step 1: Compile all services
        log "Step 1/4: Compiling all services ..."
        compile_all

        # Step 2: Sync binaries + configs
        log "Step 2/4: Syncing to server ..."
        sync_all

        # Step 3: Stop → Start
        log "Step 3/4: Restarting services ..."
        remote_stop
        remote_start

        # Step 4: Done
        echo ""
        ok "🎉 Full deployment complete!"
        echo ""
        echo "  Memory Console: https://c6.openledger.vn"
        echo "  Memory API:     https://c6.openledger.vn/v1/"
        echo "  Cognee API:     http://${DEV_SERVER}:8000"
        echo "  Graphiti API:   http://${DEV_SERVER}:8001"
        echo "  Zep API:        http://${DEV_SERVER}:8002"
        echo "  OpenViking API: http://${DEV_SERVER}:1933"
        echo "  Neo4j Browser:  http://${DEV_SERVER}:7474"
        echo ""
        ;;
esac
