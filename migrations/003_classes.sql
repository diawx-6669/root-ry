-- ============================================================
--  003 — классы и учитель
--
--  Без учителя продукт остаётся приложением для одиночки. Ученик
--  занимается сам, и никто не видит, что половина класса третью
--  неделю валит одну и ту же тему.
--
--  Появляется роль учителя, класс с кодом приглашения и связь
--  «ученик — класс». Тепловая карта и выгрузка считаются на лету из
--  attempts и topic_mastery: отдельных агрегатов нет намеренно, иначе
--  они рано или поздно разъедутся с журналом ответов.
--
--  Идемпотентна: повторный запуск ничего не ломает.
-- ============================================================

-- Роль учителя. Отдельный флаг, а не is_admin: администратор чинит
-- систему, учитель ведёт класс, и путать эти права опасно — учителю
-- незачем видеть чужие классы и править чужие данные.
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_teacher BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS classes (
    id         BIGSERIAL   PRIMARY KEY,
    -- Код приглашения: ученик вводит его один раз при вступлении.
    -- Короткий и без похожих символов — его диктуют вслух у доски.
    code       TEXT        NOT NULL UNIQUE,
    name       TEXT        NOT NULL,
    teacher_id BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    archived   BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_classes_teacher ON classes(teacher_id) WHERE NOT archived;

CREATE TABLE IF NOT EXISTS class_members (
    class_id  BIGINT      NOT NULL REFERENCES classes(id) ON DELETE CASCADE,
    user_id   BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (class_id, user_id)
);

-- Ученик может состоять в нескольких классах (русский язык и
-- подготовка к экзамену — разные группы у разных учителей).
CREATE INDEX IF NOT EXISTS idx_class_members_user ON class_members(user_id);
