-- =============================================================================
-- VNP Memory — PostgreSQL Init Script
-- Creates separate databases for each service sharing the PostgreSQL instance.
-- This script runs automatically on first container start.
-- =============================================================================

-- Enable pgvector extension on the default database
CREATE EXTENSION IF NOT EXISTS vector;

-- Cognee database
SELECT 'CREATE DATABASE cognee_db'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'cognee_db')\gexec

-- Zep database
SELECT 'CREATE DATABASE zep_db'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'zep_db')\gexec

-- KGS Platform database
SELECT 'CREATE DATABASE ba_agent_db'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'ba_agent_db')\gexec

-- Enable pgvector on each service database
\c cognee_db
CREATE EXTENSION IF NOT EXISTS vector;

\c zep_db
CREATE EXTENSION IF NOT EXISTS vector;

\c ba_agent_db
CREATE EXTENSION IF NOT EXISTS vector;
