-- ============================================================
--  007 — аватарки и значки: эмодзи → идентификаторы
--
--  Аватарки и значки хранились в базе как эмодзи ("🐱", "🏆"), а
--  рисовал их шрифт операционной системы: одна и та же награда
--  выглядела по-разному в Windows, на телефоне и на проекторе.
--  Теперь у каждой награды короткий идентификатор ("cat", "trophy"),
--  которому соответствует файл в static/img/.
--
--  Код умеет читать оба формата (internal/store/legacy.go), поэтому
--  миграция не обязательна и ничего не чинит — она просто приводит
--  данные к тому же виду, в каком их пишет новый код. После неё
--  содержимое базы совпадает с тем, что ждёт приложение.
--
--  Повторный запуск безопасен: после первого прохода старых значений
--  не остаётся, условия WHERE перестают совпадать и строки не
--  трогаются.
-- ============================================================

-- Всё одной транзакцией: временные таблицы соответствий живут до COMMIT,
-- а данные либо переводятся целиком, либо не трогаются вовсе.
BEGIN;

-- ── Соответствия ─────────────────────────────────────────────────────
--
-- Списки разные, потому что пространства имён разные: «🌈» среди
-- аватарок — мифическая радуга, среди значков — знак уровня C2, а «⭐»
-- значка и «🌟» аватарки оба зовутся star, но лежат в разных колонках.

CREATE TEMP TABLE avatar_map(old TEXT PRIMARY KEY, new TEXT) ON COMMIT DROP;
INSERT INTO avatar_map VALUES
    ('🐱','cat'), ('🐶','dog'), ('🦊','fox'), ('🐼','panda'), ('🐨','koala'),
    ('🐻','bear'), ('🐸','frog'), ('🦁','lion'), ('🐯','tiger'),
    ('🦄','unicorn'), ('🐉','dragon'), ('🦋','butterfly'), ('🦚','peacock'),
    ('🦜','parrot'), ('🦩','flamingo'), ('🐬','dolphin'),
    ('🧙','wizard'), ('🧛','vampire'), ('🦸','hero'), ('🧝','elf'), ('🧜','mermaid'),
    ('👑','crown'), ('🌟','star'), ('💫','comet'),
    ('🌈','rainbow');

CREATE TEMP TABLE badge_map(old TEXT PRIMARY KEY, new TEXT) ON COMMIT DROP;
INSERT INTO badge_map VALUES
    ('📚','book'), ('✏️','pencil'), ('✏','pencil'), ('📝','notepad'), ('🎒','backpack'),
    ('⭐','star'), ('🔥','fire'), ('💡','bulb'),
    ('🏆','trophy'), ('💎','diamond'), ('👑','crown'), ('🌳','tree'),
    ('🔰','rank_a1'), ('📗','rank_a2'), ('📘','rank_b1'),
    ('🏅','rank_b2'), ('🔮','rank_c1'), ('🦉','rank_c2'),
    -- До миграции 006 значком C2 была радуга; у кого она осталась в
    -- значках — это именно уровень, а не аватарка.
    ('🌈','rank_c2'),
    -- «🎓» выдавался демо-аккаунту при первом запуске и пары в новом
    -- каталоге не имеет; ближе всего значок за пройденное дерево.
    ('🎓','tree');

-- ── Инвентарь аватарок ───────────────────────────────────────────────
-- DISTINCT нужен на случай, когда в списке уже лежат и «⭐», и «star»:
-- без него в профиле появились бы две одинаковые награды.

UPDATE users u
SET avatars = COALESCE((
        SELECT jsonb_agg(DISTINCT COALESCE(m.new, a.val))
        FROM jsonb_array_elements_text(u.avatars) AS a(val)
        LEFT JOIN avatar_map m ON m.old = a.val
    ), '[]'::jsonb)
WHERE EXISTS (
    SELECT 1
    FROM jsonb_array_elements_text(u.avatars) AS a(val)
    JOIN avatar_map m ON m.old = a.val
);

-- ── Выбранная аватарка ───────────────────────────────────────────────

UPDATE users u
SET active_avatar = m.new
FROM avatar_map m
WHERE m.old = u.active_avatar;

-- ── Значки ───────────────────────────────────────────────────────────

UPDATE users u
SET badges = COALESCE((
        SELECT jsonb_agg(DISTINCT COALESCE(m.new, b.val))
        FROM jsonb_array_elements_text(u.badges) AS b(val)
        LEFT JOIN badge_map m ON m.old = b.val
    ), '[]'::jsonb)
WHERE EXISTS (
    SELECT 1
    FROM jsonb_array_elements_text(u.badges) AS b(val)
    JOIN badge_map m ON m.old = b.val
);

-- ── Награды промокодов ───────────────────────────────────────────────
-- В таблице промокодов лежат имена выдаваемых наград. Промокод с
-- эмодзи выдал бы предмет, которого нет в каталоге, и в профиле
-- получилась бы пустая ячейка.

UPDATE promos p
SET avatar_name = m.new
FROM avatar_map m
WHERE m.old = p.avatar_name;

UPDATE promos p
SET badge_name = m.new
FROM badge_map m
WHERE m.old = p.badge_name;

COMMIT;
