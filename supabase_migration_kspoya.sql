-- ══════════════════════════════════════════════════════════════
--  КСПОЯ — миграция Supabase (v2)
--  Запустить в Supabase Dashboard → SQL Editor ПОСЛЕ schema.sql
--
--  Заменяет supabase_migration_kspoя.sql (тот файл использовал
--  кириллическую «я» в именах таблиц и ссылался на несуществующую
--  таблицу badges — его выполнять не нужно).
--
--  Банк из 120 вопросов хранится в коде (internal/kspoya/bank.go),
--  а не в БД: правильные ответы не должны быть доступны через API.
-- ══════════════════════════════════════════════════════════════

-- Попытки прохождения теста.
-- question_ids — 40 идентификаторов вопросов В ПОРЯДКЕ ВЫДАЧИ.
-- Именно сервер помнит, какие вопросы были заданы, поэтому подменить
-- набор или порядок на клиенте невозможно.
CREATE TABLE IF NOT EXISTS kspoya_sessions (
    id           TEXT        PRIMARY KEY,
    username     TEXT        NOT NULL,
    question_ids INTEGER[]   NOT NULL,
    started_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at   TIMESTAMPTZ NOT NULL,
    status       TEXT        NOT NULL DEFAULT 'active'
                             CHECK (status IN ('active', 'completed', 'aborted')),
    raw_score    INTEGER,     -- сколько верных ответов из 40
    percent      INTEGER,     -- индекс уровня 0..100 (с поправкой на угадывание)
    level_key    TEXT,        -- A1..C2
    finished_at  TIMESTAMPTZ
);

-- Не больше одной активной попытки на пользователя.
CREATE UNIQUE INDEX IF NOT EXISTS kspoya_sessions_active_user
    ON kspoya_sessions (username)
    WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_kspoya_sessions_user
    ON kspoya_sessions (username, finished_at DESC);

-- Если ранее была запущена старая миграция с кириллическими именами,
-- эти таблицы можно удалить — код их не использует:
--   DROP TABLE IF EXISTS "kspoя_sessions", "kspoя_questions", "kspoя_results";
--   ALTER TABLE users DROP COLUMN IF EXISTS "active_kspoя_badge";
