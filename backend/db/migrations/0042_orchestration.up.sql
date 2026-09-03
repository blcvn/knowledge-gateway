-- Orchestration: Actions, Leases, Signals, Checkpoints, Sentinels, Routines, Sketches, Crystals
CREATE TABLE IF NOT EXISTS actions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       TEXT NOT NULL,
    project         TEXT NOT NULL DEFAULT '',
    agent_id        TEXT,
    title           TEXT NOT NULL,
    description     TEXT,
    status          TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','active','blocked','done','cancelled','failed')),
    priority        INTEGER NOT NULL DEFAULT 50 CHECK (priority BETWEEN 0 AND 100),
    requires        TEXT[] NOT NULL DEFAULT '{}',
    conflicts_with  TEXT[] NOT NULL DEFAULT '{}',
    tags            TEXT[] NOT NULL DEFAULT '{}',
    result          TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS leases (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    action_id   UUID NOT NULL REFERENCES actions(id) ON DELETE CASCADE,
    agent_id    TEXT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','expired','released')),
    acquired_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL,
    renewed_at  TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS signals (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT NOT NULL,
    from_agent  TEXT NOT NULL,
    to_agent    TEXT NOT NULL,
    signal_type TEXT NOT NULL CHECK (signal_type IN ('handoff','update','cancel','request','response','alert')),
    content     TEXT NOT NULL,
    thread_id   TEXT,
    reply_to    UUID REFERENCES signals(id),
    is_read     BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '7 days'),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS routines (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT NOT NULL,
    project     TEXT NOT NULL DEFAULT '',
    name        TEXT NOT NULL,
    description TEXT,
    steps       JSONB NOT NULL DEFAULT '[]',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS checkpoints (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT NOT NULL,
    project     TEXT NOT NULL DEFAULT '',
    agent_id    TEXT,
    action_id   UUID REFERENCES actions(id),
    title       TEXT NOT NULL,
    description TEXT,
    status      TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','rejected','expired')),
    approved_by TEXT,
    rejected_by TEXT,
    reason      TEXT,
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '24 hours'),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS sentinels (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    TEXT NOT NULL,
    name         TEXT NOT NULL,
    condition    JSONB NOT NULL,  -- {type, target, value}
    action_id    UUID REFERENCES actions(id),
    signal_to    TEXT,
    status       TEXT NOT NULL DEFAULT 'watching' CHECK (status IN ('watching','triggered','expired')),
    expires_at   TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '30 days'),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    triggered_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS sketches (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT NOT NULL,
    project     TEXT NOT NULL DEFAULT '',
    session_id  TEXT,
    title       TEXT NOT NULL,
    action_ids  TEXT[] NOT NULL DEFAULT '{}',
    status      TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','promoted','expired')),
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '72 hours'),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS crystals (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        TEXT NOT NULL,
    source_action_ids TEXT[] NOT NULL DEFAULT '{}',
    narrative        TEXT,
    key_outcomes     TEXT[] NOT NULL DEFAULT '{}',
    files_affected   TEXT[] NOT NULL DEFAULT '{}',
    lessons          TEXT[] NOT NULL DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_actions_tenant_status ON actions(tenant_id, status, project);
CREATE INDEX idx_leases_action ON leases(action_id, status) WHERE status = 'active';
CREATE INDEX idx_leases_expires ON leases(expires_at) WHERE status = 'active';
CREATE INDEX idx_signals_to_agent ON signals(tenant_id, to_agent, is_read);
CREATE INDEX idx_signals_expires ON signals(expires_at);
CREATE INDEX idx_checkpoints_status ON checkpoints(tenant_id, status);
CREATE INDEX idx_sentinels_watching ON sentinels(status) WHERE status = 'watching';
