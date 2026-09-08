-- ============================================================
--  006 — уроки дерева грамматики
--
--  Бывший supabase_migration_lessons.sql из корня проекта; переехал
--  в migrations/ по той же причине, что и 005.
-- ============================================================

UPDATE users
SET nickname = replace(
                 replace(
                   replace(
                     replace(
                       replace(nickname, '&lt;', '<'),
                     '&gt;', '>'),
                   '&quot;', '"'),
                 '&#39;', ''''),
               '&amp;', '&')
WHERE nickname LIKE '%&lt;%'
   OR nickname LIKE '%&gt;%'
   OR nickname LIKE '%&quot;%'
   OR nickname LIKE '%&#39;%'
   OR nickname LIKE '%&amp;%';

-- ── 2. Пройденные темы: чистим мусор ─────────────────────────────────
-- /api/topic/complete принимал любую строку, поэтому в completed_topics
-- могли осесть произвольные значения, накрученные ради XP. Оставляем
-- только идентификаторы из реестра internal/topics.
UPDATE users
SET completed_topics = COALESCE((
        SELECT jsonb_agg(t)
        FROM jsonb_array_elements_text(completed_topics) AS t
        WHERE t IN (
            'fon-zvuki','fon-soglasnye','fon-udarenie','fon-intonaciya','fon-slog','fon-razbor',
            'graf-kirillica','graf-bukvy-zvuki','graf-propisnye','graf-alfavit',
            'orf-glasnye','orf-soglasnye','orf-znaki','orf-shipyashchie','orf-pristavki',
            'orf-suffiksy','orf-okonchaniya','orf-ne-ni','orf-n-nn','orf-slitno',
            'orf-defis','orf-razbor',
            'mrf-morfemy','mrf-odnokorennye','mrf-razbor',
            'slv-sposoby','slv-razbor','slv-cepochka',
            'lex-znachenie','lex-otnosheniya','lex-proishozhdenie','lex-aktivnaya',
            'lex-frazeologiya','lex-okraska','lex-razbor',
            'mor-sushestvitelnoe','mor-prilagatelnoe','mor-chislitelnoe','mor-mestoimenie',
            'mor-glagol','mor-prichastie','mor-deeprichastie','mor-narechie','mor-predlog',
            'mor-soyuz','mor-chastica','mor-mezhdometie','mor-razbor',
            'sin-slovosochetanie','sin-prostoe','sin-slozhnoe','sin-pryamaya-rech',
            'sin-dialog','sin-citirovanie','sin-razbor',
            'pun-konec','pun-zapyataya','pun-tochka-zapyataya','pun-dvoetochie','pun-tire',
            'pun-kavychki','pun-skobki','pun-avtorskaya',
            'sti-stili','sti-sredstva','sti-oshibki',
            'kul-normy','kul-etiket','kul-pravilnost',
            'tex-priznaki','tex-tema-ideya','tex-tipy-rechi','tex-svyazi','tex-abzac'
        )
    ), '[]'::jsonb)
WHERE jsonb_array_length(completed_topics) > 0;

-- ── 3. Значок C2 в КСПОЯ ─────────────────────────────────────────────
-- Раньше за C2 давали 🌈 — ту же радугу, что выпадает мифической аватаркой
-- из кейса. Значок высшего уровня переехал на 🦉; у тех, кто уже сдал C2,
-- меняем его в инвентаре.
UPDATE users u
SET badges = (
        SELECT jsonb_agg(CASE WHEN b = '🌈' THEN '🦉' ELSE b END)
        FROM jsonb_array_elements_text(u.badges) AS b
    )
WHERE u.badges ? '🌈'
  AND EXISTS (
      SELECT 1 FROM kspoya_sessions s
      WHERE s.username = u.username AND s.status = 'completed' AND s.level_key = 'C2'
  );

-- ── 4. Ежедневный бонус ──────────────────────────────────────────────
-- Бонусов было два: молчаливый +10 за любой заход и кнопка в магазине.
-- Автоматический убран в коде, отдельная миграция данных не нужна.
