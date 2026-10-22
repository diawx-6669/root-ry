-- ============================================================
--  RootRy — PostgreSQL schema for Supabase
--  Run once in Supabase SQL Editor (or psql)
-- ============================================================

-- ── Users ────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS users (
    id               BIGSERIAL PRIMARY KEY,
    username         TEXT        NOT NULL UNIQUE,
    nickname         TEXT        NOT NULL,
    password_hash    TEXT        NOT NULL,
    balance          INTEGER     NOT NULL DEFAULT 500,
    xp               INTEGER     NOT NULL DEFAULT 0,
    streak           INTEGER     NOT NULL DEFAULT 0,
    last_login       DATE,
    last_nick_change DATE,
    is_admin         BOOLEAN     NOT NULL DEFAULT FALSE,
    -- Arrays stored as JSONB — simplest approach in Go (no lib/pq TEXT[] needed)
    badges           JSONB       NOT NULL DEFAULT '[]',
    avatars          JSONB       NOT NULL DEFAULT '["🐱"]',
    active_avatar    TEXT        NOT NULL DEFAULT '🐱',
    completed_topics JSONB       NOT NULL DEFAULT '[]',
    promo_used       JSONB       NOT NULL DEFAULT '[]',
    favorite_games   JSONB       NOT NULL DEFAULT '[]',
    daily_tasks_date DATE,
    daily_tasks_done INTEGER     NOT NULL DEFAULT 0,
    games_won_today  INTEGER     NOT NULL DEFAULT 0,
    games_won_types  JSONB       NOT NULL DEFAULT '[]',
    last_daily_claim DATE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Promo codes ──────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS promos (
    code         TEXT    PRIMARY KEY,
    reward       TEXT    NOT NULL,   -- coins | xp | badge | avatar | admin
    value        INTEGER NOT NULL DEFAULT 0,
    badge_name   TEXT    NOT NULL DEFAULT '',
    avatar_name  TEXT    NOT NULL DEFAULT '',
    uses         INTEGER NOT NULL DEFAULT -1,  -- -1 = unlimited
    used_count   INTEGER NOT NULL DEFAULT 0
);

-- Seed default promo codes (idempotent)
INSERT INTO promos (code, reward, value, uses) VALUES
    ('MEGACOINS', 'coins', 10000000, -1),
    ('777',       'coins', 10000000, -1)
ON CONFLICT (code) DO NOTHING;
-- Промокод на права администратора здесь больше не создаётся: код лежал
-- в открытых исходниках. Сервер создаёт его сам из переменной ADMIN_PROMO.

-- ── Game results ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS game_results (
    id           BIGSERIAL   PRIMARY KEY,
    user_id      BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    game_type    TEXT        NOT NULL,
    score        INTEGER     NOT NULL DEFAULT 0,
    xp_earned    INTEGER     NOT NULL DEFAULT 0,
    coins_earned INTEGER     NOT NULL DEFAULT 0,
    played_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_game_results_user ON game_results(user_id);

-- ── Test results ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS test_results (
    id           BIGSERIAL   PRIMARY KEY,
    user_id      BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    score        INTEGER     NOT NULL DEFAULT 0,
    passed       BOOLEAN     NOT NULL DEFAULT FALSE,
    level        TEXT        NOT NULL DEFAULT '',
    badge_earned TEXT        NOT NULL DEFAULT '',
    played_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Case (loot-box) results ───────────────────────────────────
CREATE TABLE IF NOT EXISTS case_results (
    id           BIGSERIAL   PRIMARY KEY,
    user_id      BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    case_type    TEXT        NOT NULL,
    item_emoji   TEXT        NOT NULL,
    item_rarity  TEXT        NOT NULL,
    is_duplicate BOOLEAN     NOT NULL DEFAULT FALSE,
    compensation INTEGER     NOT NULL DEFAULT 0,
    played_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_case_results_user ON case_results(user_id);
