-- ============================================================
--  008 — вход через Google
--
--  Привязка Google-аккаунта к ученику. Отдельная таблица, а не столбец
--  в users: основной SELECT пользователя не меняется, а у одного
--  ученика в будущем может быть больше одного способа входа.
--
--  google_sub — неизменный идентификатор аккаунта Google (поле "sub"
--  в ID-токене). Почту ключом не берём: её можно сменить.
--
--  Повторный запуск безопасен.
-- ============================================================

CREATE TABLE IF NOT EXISTS google_accounts (
    google_sub TEXT        PRIMARY KEY,
    user_id    BIGINT      NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    email      TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
