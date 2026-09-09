-- ══════════════════════════════════════════════════════════════
--  КСПОЯ — Миграция Supabase
--  Запусти в Supabase Dashboard → SQL Editor
-- ══════════════════════════════════════════════════════════════

-- 1. Таблица вопросов (100 вопросов по грамматике)
CREATE TABLE IF NOT EXISTS kspoя_questions (
    id          SERIAL PRIMARY KEY,
    question    TEXT        NOT NULL,
    option_a    TEXT        NOT NULL,
    option_b    TEXT        NOT NULL,
    option_c    TEXT        NOT NULL,
    option_d    TEXT        NOT NULL,
    correct_idx INT         NOT NULL CHECK (correct_idx BETWEEN 0 AND 3),
    category    TEXT        DEFAULT 'grammar',
    created_at  TIMESTAMPTZ DEFAULT now()
);

-- 2. Таблица сессий тестирования (1 активная на пользователя)
CREATE TABLE IF NOT EXISTS kspoя_sessions (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      TEXT        NOT NULL,          -- username
    question_ids INT[]       NOT NULL,          -- 40 случайных ID из kspoя_questions
    started_at   TIMESTAMPTZ DEFAULT now(),
    expires_at   TIMESTAMPTZ NOT NULL,          -- started_at + 40 min
    status       TEXT        NOT NULL DEFAULT 'active'
                             CHECK (status IN ('active','completed','cancelled')),
    score        INT,
    level_key    TEXT,
    finished_at  TIMESTAMPTZ
);

-- Только 1 активная сессия на пользователя
CREATE UNIQUE INDEX IF NOT EXISTS kspoя_sessions_active_user
    ON kspoя_sessions (user_id)
    WHERE status = 'active';

-- 3. Таблица результатов (история)
CREATE TABLE IF NOT EXISTS kspoя_results (
    id          SERIAL      PRIMARY KEY,
    user_id     TEXT        NOT NULL,
    score       INT         NOT NULL,
    level_key   TEXT        NOT NULL,
    coins_given INT         NOT NULL DEFAULT 0,
    finished_at TIMESTAMPTZ DEFAULT now()
);

-- 4. КСПОЯ-значки в таблице badges
-- Добавляем колонку source чтобы отличать ксп-значки от промо
ALTER TABLE badges ADD COLUMN IF NOT EXISTS source TEXT DEFAULT 'promo';
-- source = 'kspoя' — нельзя получить через промокод

-- Если таблицы badges ещё нет — создаём
CREATE TABLE IF NOT EXISTS badges (
    id          SERIAL      PRIMARY KEY,
    code        TEXT        NOT NULL UNIQUE,   -- 'kspoя_c2', 'kspoя_c1', ...
    label       TEXT        NOT NULL,
    description TEXT,
    rarity      TEXT        NOT NULL,          -- 'rainbow','mythic','legendary','epic','rare','common'
    source      TEXT        NOT NULL DEFAULT 'promo', -- 'kspoя' = защищённый
    image_path  TEXT,                          -- путь к PNG в /static/badges/
    created_at  TIMESTAMPTZ DEFAULT now()
);

-- 5. Вставляем 6 КСПОЯ-значков
INSERT INTO badges (code, label, description, rarity, source, image_path) VALUES
  ('kspoя_c2', 'C2 — Мастерство',       'Радужный значок КСПОЯ. 38–40 из 40.',  'rainbow',   'kspoя', 'badges/kspoя_c2.png'),
  ('kspoя_c1', 'C1 — Продвинутый',      'Мифический значок КСПОЯ. 32–37 из 40.','mythic',    'kspoя', 'badges/kspoя_c1.png'),
  ('kspoя_b2', 'B2 — Выше среднего',    'Легендарный значок КСПОЯ. 25–31/40.',  'legendary', 'kspoя', 'badges/kspoя_b2.png'),
  ('kspoя_b1', 'B1 — Средний',          'Эпический значок КСПОЯ. 15–24 из 40.', 'epic',      'kspoя', 'badges/kspoя_b1.png'),
  ('kspoя_a2', 'A2 — Элементарный',     'Редкий значок КСПОЯ. 5–14 из 40.',    'rare',      'kspoя', 'badges/kspoя_a2.png'),
  ('kspoя_a1', 'A1 — Начальный',        'Обычный значок КСПОЯ. 0–4 из 40.',    'common',    'kspoя', 'badges/kspoя_a1.png')
ON CONFLICT (code) DO NOTHING;

-- 6. Таблица пользовательских значков (если нет)
CREATE TABLE IF NOT EXISTS user_badges (
    id         SERIAL      PRIMARY KEY,
    user_id    TEXT        NOT NULL,
    badge_code TEXT        NOT NULL REFERENCES badges(code),
    earned_at  TIMESTAMPTZ DEFAULT now(),
    UNIQUE(user_id, badge_code)
);

-- 7. Поле active_kspoя_badge в users (выбранный КСПОЯ-значок для профиля)
ALTER TABLE users ADD COLUMN IF NOT EXISTS active_kspoя_badge TEXT DEFAULT NULL;

-- Индексы
CREATE INDEX IF NOT EXISTS idx_kspoя_results_user ON kspoя_results(user_id);
CREATE INDEX IF NOT EXISTS idx_user_badges_user   ON user_badges(user_id);
CREATE INDEX IF NOT EXISTS idx_kspoя_sessions_user ON kspoя_sessions(user_id);