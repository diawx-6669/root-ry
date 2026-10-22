-- ============================================================
--  001 — модель знаний ученика
--
--  До этой миграции система помнила только факт «тема пройдена».
--  Ученик, решивший 8 заданий из 8, и ученик, доклацавший 4 из 8,
--  выглядели для базы одинаково, а забывание не моделировалось
--  вообще: пройденная в сентябре тема к декабрю всё ещё считалась
--  освоенной.
--
--  Появляется три таблицы:
--    attempts      — журнал каждого ответа, откуда бы он ни пришёл;
--    topic_mastery — коробка Лейтнера по теме: когда вернуть ученика;
--    item_state    — состояние конкретного задания для тетради ошибок.
--
--  Идемпотентна: повторный запуск ничего не ломает.
-- ============================================================

-- ── Журнал ответов ───────────────────────────────────────────
-- Пишется на каждый ответ и никогда не обновляется. Это сырьё:
-- из него считается и прогресс ученика, и статистика для
-- исследования (какие темы валит весь класс).
CREATE TABLE IF NOT EXISTS attempts (
    id          BIGSERIAL   PRIMARY KEY,
    user_id     BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    topic_id    TEXT        NOT NULL,   -- идентификатор из internal/topics
    item_id     TEXT        NOT NULL,   -- topic_id + '#' + номер задания
    source      TEXT        NOT NULL,   -- lesson | game | kspoya | review
    correct     BOOLEAN     NOT NULL,
    hints_used  INTEGER     NOT NULL DEFAULT 0,
    time_ms     INTEGER     NOT NULL DEFAULT 0,
    answered_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_attempts_user_topic
    ON attempts(user_id, topic_id, answered_at DESC);
CREATE INDEX IF NOT EXISTS idx_attempts_user_time
    ON attempts(user_id, answered_at DESC);
-- Для исследовательских выгрузок: срез по теме через весь класс.
CREATE INDEX IF NOT EXISTS idx_attempts_topic_time
    ON attempts(topic_id, answered_at DESC);

-- ── Владение темой ───────────────────────────────────────────
-- box  — коробка Лейтнера 1..5, интервалы 1 / 3 / 7 / 16 / 35 дней.
-- due_on — дата, когда тему нужно показать снова. Если она в прошлом,
--          тема «просрочена» и попадает в план повторения на сегодня.
-- streak_days — сколько РАЗНЫХ дней подряд ученик отвечает верно.
--          Именно дней, а не ответов: восемь верных подряд за один
--          вечер не доказывают, что тема осталась в голове назавтра.
CREATE TABLE IF NOT EXISTS topic_mastery (
    user_id     BIGINT  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    topic_id    TEXT    NOT NULL,
    box         INTEGER NOT NULL DEFAULT 1,
    streak_days INTEGER NOT NULL DEFAULT 0,
    lapses      INTEGER NOT NULL DEFAULT 0,
    correct     INTEGER NOT NULL DEFAULT 0,
    total       INTEGER NOT NULL DEFAULT 0,
    due_on      DATE    NOT NULL,
    last_seen   DATE,
    first_done  DATE,
    -- Дата, начиная с которой за эту тему снова можно получить награду.
    --
    -- Отдельная колонка, а не due_on, потому что это разные вещи. due_on
    -- двигается на каждый ответ и говорит, когда ПОКАЗАТЬ тему. Награду же
    -- нельзя привязывать к ответам: ученик прошёл бы урок, получил монеты,
    -- тут же прошёл снова и получил ещё. next_reward_on двигается только в
    -- момент выдачи награды, ровно на длину текущего интервала.
    --
    -- NULL означает «награда за тему ещё ни разу не выдавалась», то есть
    -- первое прохождение впереди.
    next_reward_on DATE,
    PRIMARY KEY (user_id, topic_id)
);

CREATE INDEX IF NOT EXISTS idx_topic_mastery_due
    ON topic_mastery(user_id, due_on);

-- ── Состояние задания (тетрадь ошибок) ───────────────────────
-- Задание попадает в тетрадь после первой ошибки и уходит из неё,
-- когда ученик ответил верно дважды в РАЗНЫЕ дни. Одного верного
-- ответа сразу после показа разбора недостаточно: это память о
-- только что прочитанном, а не знание.
-- box и streak_days считаются тем же кодом, что и для темы: одно правило
-- продвижения на оба уровня, чтобы «закрыто» и «освоено» не разъезжались.
-- correct/total нужны не ученику, а учителю и исследованию: по ним видно,
-- какое конкретно задание проваливает весь класс.
CREATE TABLE IF NOT EXISTS item_state (
    user_id      BIGINT  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    item_id      TEXT    NOT NULL,
    topic_id     TEXT    NOT NULL,
    box          INTEGER NOT NULL DEFAULT 0,
    streak_days  INTEGER NOT NULL DEFAULT 0,
    wrong_count  INTEGER NOT NULL DEFAULT 0,
    correct      INTEGER NOT NULL DEFAULT 0,
    total        INTEGER NOT NULL DEFAULT 0,
    resolved     BOOLEAN NOT NULL DEFAULT FALSE,
    last_wrong   DATE,
    last_seen    DATE,
    PRIMARY KEY (user_id, item_id)
);

-- Тетрадь ошибок читает именно этот срез: незакрытые задания ученика.
CREATE INDEX IF NOT EXISTS idx_item_state_open
    ON item_state(user_id, resolved, last_wrong DESC);
