-- ============================================================
--  005 — КСПОЯ: попытки теста, античит, рейтинг
--
--  Раньше этот файл лежал в корне как supabase_migration_kspoya.sql и
--  запускался руками отдельно от migrations/. Свежая база, поднятая по
--  инструкции «schema.sql + migrations/*», оставалась без таблицы
--  kspoya_sessions: тесты и рейтинг молча отдавали пустые списки, а в
--  логах копились «relation kspoya_sessions does not exist». Теперь это
--  обычная миграция, и порядок один и тот же везде.
--
--  Банк из 120 вопросов хранится в коде (internal/kspoya/bank.go),
--  а не в БД: правильные ответы не должны быть доступны через API.
-- ============================================================

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

-- Ответы ученика сохраняются, чтобы разбор попытки можно было открыть позже
-- из истории. Порядок совпадает с question_ids.
ALTER TABLE kspoya_sessions
    ADD COLUMN IF NOT EXISTS answers INTEGER[];

-- ── Античит: предупреждения и блокировка ─────────────────────────────
-- Первое нарушение (уход со вкладки или выход из полноэкранного режима)
-- аннулирует попытку и даёт предупреждение, второе — блокирует КСПОЯ
-- на 24 часа. После блокировки счётчик обнуляется.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS kspoya_warnings INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS kspoya_ban_until TIMESTAMPTZ;

-- Рейтинг КСПОЯ: лучшая попытка каждого пользователя.
-- При равном балле выше тот, кто прошёл раньше.
CREATE INDEX IF NOT EXISTS idx_kspoya_sessions_rating
    ON kspoya_sessions (raw_score DESC, finished_at ASC)
    WHERE status = 'completed';

-- Если ранее была запущена старая миграция с кириллическими именами,
-- эти таблицы можно удалить — код их не использует:
--   DROP TABLE IF EXISTS "kspoя_sessions", "kspoя_questions", "kspoя_results";
--   ALTER TABLE users DROP COLUMN IF EXISTS "active_kspoя_badge";
