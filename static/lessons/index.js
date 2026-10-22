/* ═══════════════════════════════════════════════════════════════════
   RootRy — индекс уроков дерева грамматики.

   74 темы, по одной на каждый «узел-тему» дерева. Индекс лёгкий: его
   грузит и страница дерева, и плеер урока. Само содержание урока лежит
   в файле своего раздела (s1.js … s12.js) и подгружается только тогда,
   когда урок открывают.

   Идентификаторы обязаны совпадать с internal/topics/topics.go — сервер
   принимает /api/topic/complete только для тем из своего реестра.

   icon — класс Font Awesome, а не эмодзи: иконка должна одинаково
   выглядеть на всех системах и наследовать цвет темы.
   ═══════════════════════════════════════════════════════════════════ */

const LESSON_INDEX = [
  // ── 1. Фонетика ───────────────────────────────────────────────────
  { id:'fon-zvuki',        file:'s1',  node:'Звуки речи',                    section:'1. ФОНЕТИКА',          title:'Звуки речи',                icon:'fa-volume-high',   xp:50 },
  { id:'fon-soglasnye',    file:'s1',  node:'Характеристики согласных',      section:'1. ФОНЕТИКА',          title:'Характеристики согласных',  icon:'fa-sliders',       xp:60 },
  { id:'fon-udarenie',     file:'s1',  node:'Ударение',                      section:'1. ФОНЕТИКА',          title:'Ударение',                  icon:'fa-bullhorn',      xp:50 },
  { id:'fon-intonaciya',   file:'s1',  node:'Интонация',                     section:'1. ФОНЕТИКА',          title:'Интонация',                 icon:'fa-wave-square',   xp:40 },
  { id:'fon-slog',         file:'s1',  node:'Слог',                          section:'1. ФОНЕТИКА',          title:'Слог',                      icon:'fa-divide',        xp:40 },
  { id:'fon-razbor',       file:'s1',  node:'Фонетический разбор',           section:'1. ФОНЕТИКА',          title:'Фонетический разбор',       icon:'fa-microscope',    xp:70 },

  // ── 2. Графика и алфавит ──────────────────────────────────────────
  { id:'graf-kirillica',   file:'s2',  node:'Кириллица',                     section:'2. ГРАФИКА И АЛФАВИТ', title:'Кириллица',                 icon:'fa-scroll',        xp:40 },
  { id:'graf-bukvy-zvuki', file:'s2',  node:'Буквы и звуки',                 section:'2. ГРАФИКА И АЛФАВИТ', title:'Буквы и звуки',             icon:'fa-font',          xp:60 },
  { id:'graf-propisnye',   file:'s2',  node:'Прописные / строчные буквы',    section:'2. ГРАФИКА И АЛФАВИТ', title:'Прописные и строчные',      icon:'fa-heading',       xp:40 },
  { id:'graf-alfavit',     file:'s2',  node:'Алфавитный порядок',            section:'2. ГРАФИКА И АЛФАВИТ', title:'Алфавитный порядок',        icon:'fa-list-ol',       xp:40 },

  // ── 3. Орфография ─────────────────────────────────────────────────
  { id:'orf-glasnye',      file:'s3',  node:'Правописание гласных',          section:'3. ОРФОГРАФИЯ',        title:'Правописание гласных',      icon:'fa-spell-check',   xp:70 },
  { id:'orf-soglasnye',    file:'s3',  node:'Правописание согласных',        section:'3. ОРФОГРАФИЯ',        title:'Правописание согласных',    icon:'fa-pen',           xp:60 },
  { id:'orf-znaki',        file:'s3',  node:'Ь и Ъ',                         section:'3. ОРФОГРАФИЯ',        title:'Ь и Ъ',                     icon:'fa-pen-nib',       xp:60 },
  { id:'orf-shipyashchie', file:'s3',  node:'Правила после шипящих',         section:'3. ОРФОГРАФИЯ',        title:'После шипящих',             icon:'fa-wind',          xp:60 },
  { id:'orf-pristavki',    file:'s3',  node:'Приставки',                     section:'3. ОРФОГРАФИЯ',        title:'Приставки',                 icon:'fa-backward',      xp:70 },
  { id:'orf-suffiksy',     file:'s3',  node:'Суффиксы',                      section:'3. ОРФОГРАФИЯ',        title:'Суффиксы',                  icon:'fa-forward',       xp:70 },
  { id:'orf-okonchaniya',  file:'s3',  node:'Окончания',                     section:'3. ОРФОГРАФИЯ',        title:'Окончания',                 icon:'fa-flag-checkered',xp:60 },
  { id:'orf-ne-ni',        file:'s3',  node:'НЕ и НИ',                       section:'3. ОРФОГРАФИЯ',        title:'НЕ и НИ',                   icon:'fa-ban',           xp:80 },
  { id:'orf-n-nn',         file:'s3',  node:'Н и НН',                        section:'3. ОРФОГРАФИЯ',        title:'Н и НН',                    icon:'fa-bullseye',      xp:80 },
  { id:'orf-slitno',       file:'s3',  node:'Слитное / раздельное написание',section:'3. ОРФОГРАФИЯ',        title:'Слитно или раздельно',      icon:'fa-link',          xp:70 },
  { id:'orf-defis',        file:'s3',  node:'Дефисное написание',            section:'3. ОРФОГРАФИЯ',        title:'Дефисное написание',        icon:'fa-minus',         xp:60 },
  { id:'orf-razbor',       file:'s3',  node:'Орфографический разбор',        section:'3. ОРФОГРАФИЯ',        title:'Орфографический разбор',    icon:'fa-magnifying-glass', xp:50 },

  // ── 4. Морфемика ──────────────────────────────────────────────────
  { id:'mrf-morfemy',      file:'s4',  node:'Морфемы',                       section:'4. МОРФЕМИКА',         title:'Морфемы',                   icon:'fa-cubes',         xp:60 },
  // level уточняет узел, если такая подпись в дереве встречается не один раз:
  // «Однокоренные слова» есть и разделом морфемики, и подпунктом корня.
  { id:'mrf-odnokorennye', file:'s4',  node:'Однокоренные слова', level:2,   section:'4. МОРФЕМИКА',         title:'Однокоренные слова',        icon:'fa-seedling',      xp:50 },
  { id:'mrf-razbor',       file:'s4',  node:'Морфемный разбор',              section:'4. МОРФЕМИКА',         title:'Морфемный разбор',          icon:'fa-puzzle-piece',  xp:60 },

  // ── 5. Словообразование ───────────────────────────────────────────
  { id:'slv-sposoby',      file:'s5',  node:'Способы образования слов',      section:'5. СЛОВООБРАЗОВАНИЕ',  title:'Способы образования слов',  icon:'fa-industry',      xp:70 },
  { id:'slv-razbor',       file:'s5',  node:'Словообразовательный разбор',   section:'5. СЛОВООБРАЗОВАНИЕ',  title:'Словообразовательный разбор',icon:'fa-flask',        xp:60 },
  { id:'slv-cepochka',     file:'s5',  node:'Словообразовательная цепочка',  section:'5. СЛОВООБРАЗОВАНИЕ',  title:'Словообразовательная цепочка',icon:'fa-diagram-project',xp:50 },

  // ── 6. Лексикология ───────────────────────────────────────────────
  { id:'lex-znachenie',    file:'s6',  node:'Значение слова',                section:'6. ЛЕКСИКОЛОГИЯ',      title:'Значение слова',            icon:'fa-lightbulb',     xp:50 },
  { id:'lex-otnosheniya',  file:'s6',  node:'Лексические отношения',         section:'6. ЛЕКСИКОЛОГИЯ',      title:'Синонимы, антонимы, омонимы',icon:'fa-shuffle',      xp:70 },
  { id:'lex-proishozhdenie',file:'s6', node:'Происхождение слов',            section:'6. ЛЕКСИКОЛОГИЯ',      title:'Происхождение слов',        icon:'fa-earth-americas',xp:50 },
  { id:'lex-aktivnaya',    file:'s6',  node:'Активная / пассивная лексика',  section:'6. ЛЕКСИКОЛОГИЯ',      title:'Активная и пассивная лексика',icon:'fa-hourglass-half',xp:50 },
  { id:'lex-frazeologiya', file:'s6',  node:'Фразеология',                   section:'6. ЛЕКСИКОЛОГИЯ',      title:'Фразеология',               icon:'fa-comments',      xp:60 },
  { id:'lex-okraska',      file:'s6',  node:'Стилистическая окраска',        section:'6. ЛЕКСИКОЛОГИЯ',      title:'Стилистическая окраска',    icon:'fa-palette',       xp:50 },
  { id:'lex-razbor',       file:'s6',  node:'Лексический разбор',            section:'6. ЛЕКСИКОЛОГИЯ',      title:'Лексический разбор',        icon:'fa-book-open',     xp:50 },

  // ── 7. Морфология ─────────────────────────────────────────────────
  { id:'mor-sushestvitelnoe',file:'s7',node:'Имя существительное',           section:'7. МОРФОЛОГИЯ',        title:'Имя существительное',       icon:'fa-box',           xp:80 },
  { id:'mor-prilagatelnoe',file:'s7',  node:'Имя прилагательное',            section:'7. МОРФОЛОГИЯ',        title:'Имя прилагательное',        icon:'fa-brush',         xp:70 },
  { id:'mor-chislitelnoe', file:'s7',  node:'Имя числительное',              section:'7. МОРФОЛОГИЯ',        title:'Имя числительное',          icon:'fa-calculator',    xp:70 },
  { id:'mor-mestoimenie',  file:'s7',  node:'Местоимение',                   section:'7. МОРФОЛОГИЯ',        title:'Местоимение',               icon:'fa-hand-point-right',xp:70 },
  { id:'mor-glagol',       file:'s7',  node:'Глагол',                        section:'7. МОРФОЛОГИЯ',        title:'Глагол',                    icon:'fa-person-running',xp:90 },
  { id:'mor-prichastie',   file:'s7',  node:'Причастие',                     section:'7. МОРФОЛОГИЯ',        title:'Причастие',                 icon:'fa-dna',           xp:80 },
  { id:'mor-deeprichastie',file:'s7',  node:'Деепричастие',                  section:'7. МОРФОЛОГИЯ',        title:'Деепричастие',              icon:'fa-arrows-spin',   xp:70 },
  { id:'mor-narechie',     file:'s7',  node:'Наречие',                       section:'7. МОРФОЛОГИЯ',        title:'Наречие',                   icon:'fa-bolt',          xp:60 },
  { id:'mor-predlog',      file:'s7',  node:'Предлог',                       section:'7. МОРФОЛОГИЯ',        title:'Предлог',                   icon:'fa-plug',          xp:50 },
  { id:'mor-soyuz',        file:'s7',  node:'Союз',                          section:'7. МОРФОЛОГИЯ',        title:'Союз',                      icon:'fa-handshake',     xp:60 },
  { id:'mor-chastica',     file:'s7',  node:'Частица',                       section:'7. МОРФОЛОГИЯ',        title:'Частица',                   icon:'fa-wand-magic-sparkles',xp:60 },
  { id:'mor-mezhdometie',  file:'s7',  node:'7.3 Междометия',                section:'7. МОРФОЛОГИЯ',        title:'Междометие',                icon:'fa-circle-exclamation',xp:40 },
  { id:'mor-razbor',       file:'s7',  node:'Морфологический разбор',        section:'7. МОРФОЛОГИЯ',        title:'Морфологический разбор',    icon:'fa-list-check',    xp:60 },

  // ── 8. Синтаксис ──────────────────────────────────────────────────
  { id:'sin-slovosochetanie',file:'s8',node:'Словосочетание',                section:'8. СИНТАКСИС',         title:'Словосочетание',            icon:'fa-link',          xp:60 },
  { id:'sin-prostoe',      file:'s8',  node:'Простое предложение',           section:'8. СИНТАКСИС',         title:'Простое предложение',       icon:'fa-ruler',         xp:90 },
  { id:'sin-slozhnoe',     file:'s8',  node:'Сложное предложение',           section:'8. СИНТАКСИС',         title:'Сложное предложение',       icon:'fa-sitemap',       xp:90 },
  // «Прямая речь» есть и разделом синтаксиса, и подпунктом кавычек.
  { id:'sin-pryamaya-rech',file:'s8',  node:'Прямая речь', level:2,          section:'8. СИНТАКСИС',         title:'Прямая речь',               icon:'fa-comment',       xp:70 },
  { id:'sin-dialog',       file:'s8',  node:'Диалог',                        section:'8. СИНТАКСИС',         title:'Диалог',                    icon:'fa-comment-dots',  xp:40 },
  { id:'sin-citirovanie',  file:'s8',  node:'Цитирование',                   section:'8. СИНТАКСИС',         title:'Цитирование',               icon:'fa-quote-right',   xp:50 },
  { id:'sin-razbor',       file:'s8',  node:'Синтаксический разбор',         section:'8. СИНТАКСИС',         title:'Синтаксический разбор',     icon:'fa-vector-square', xp:60 },

  // ── 9. Пунктуация ─────────────────────────────────────────────────
  { id:'pun-konec',        file:'s9',  node:'Знаки конца предложения',       section:'9. ПУНКТУАЦИЯ',        title:'Знаки конца предложения',   icon:'fa-circle-dot',    xp:40 },
  { id:'pun-zapyataya',    file:'s9',  node:'Запятая',                       section:'9. ПУНКТУАЦИЯ',        title:'Запятая',                   icon:'fa-scissors',      xp:90 },
  { id:'pun-tochka-zapyataya',file:'s9',node:'Точка с запятой',              section:'9. ПУНКТУАЦИЯ',        title:'Точка с запятой',           icon:'fa-pause',         xp:50 },
  { id:'pun-dvoetochie',   file:'s9',  node:'Двоеточие',                     section:'9. ПУНКТУАЦИЯ',        title:'Двоеточие',                 icon:'fa-ellipsis-vertical',xp:70 },
  { id:'pun-tire',         file:'s9',  node:'Тире',                          section:'9. ПУНКТУАЦИЯ',        title:'Тире',                      icon:'fa-grip-lines',    xp:80 },
  { id:'pun-kavychki',     file:'s9',  node:'Кавычки',                       section:'9. ПУНКТУАЦИЯ',        title:'Кавычки',                   icon:'fa-quote-left',    xp:50 },
  { id:'pun-skobki',       file:'s9',  node:'Скобки',                        section:'9. ПУНКТУАЦИЯ',        title:'Скобки',                    icon:'fa-square',        xp:40 },
  { id:'pun-avtorskaya',   file:'s9',  node:'Авторская пунктуация',          section:'9. ПУНКТУАЦИЯ',        title:'Авторская пунктуация',      icon:'fa-pen-fancy',     xp:50 },

  // ── 10. Стилистика ────────────────────────────────────────────────
  { id:'sti-stili',        file:'s10', node:'Стили речи',                    section:'10. СТИЛИСТИКА',       title:'Стили речи',                icon:'fa-user-tie',      xp:70 },
  { id:'sti-sredstva',     file:'s10', node:'Средства выразительности',      section:'10. СТИЛИСТИКА',       title:'Средства выразительности',  icon:'fa-masks-theater', xp:80 },
  { id:'sti-oshibki',      file:'s10', node:'Речевые ошибки',                section:'10. СТИЛИСТИКА',       title:'Речевые ошибки',            icon:'fa-triangle-exclamation',xp:60 },

  // ── 11. Культура речи ─────────────────────────────────────────────
  { id:'kul-normy',        file:'s11', node:'Нормы языка',                   section:'11. КУЛЬТУРА РЕЧИ',    title:'Нормы языка',               icon:'fa-scale-balanced',xp:70 },
  { id:'kul-etiket',       file:'s11', node:'Речевой этикет',                section:'11. КУЛЬТУРА РЕЧИ',    title:'Речевой этикет',            icon:'fa-face-smile',    xp:40 },
  { id:'kul-pravilnost',   file:'s11', node:'Правильность речи',             section:'11. КУЛЬТУРА РЕЧИ',    title:'Правильность речи',         icon:'fa-check-double',  xp:50 },

  // ── 12. Текстоведение ─────────────────────────────────────────────
  { id:'tex-priznaki',     file:'s12', node:'Признаки текста',               section:'12. ТЕКСТОВЕДЕНИЕ',    title:'Признаки текста',           icon:'fa-file-lines',    xp:50 },
  { id:'tex-tema-ideya',   file:'s12', node:'Тема и идея',                   section:'12. ТЕКСТОВЕДЕНИЕ',    title:'Тема и идея',               icon:'fa-crosshairs',    xp:50 },
  { id:'tex-tipy-rechi',   file:'s12', node:'Типы речи',                     section:'12. ТЕКСТОВЕДЕНИЕ',    title:'Типы речи',                 icon:'fa-folder-tree',   xp:70 },
  { id:'tex-svyazi',       file:'s12', node:'Средства связи',                section:'12. ТЕКСТОВЕДЕНИЕ',    title:'Средства связи',            icon:'fa-code-branch',   xp:50 },
  { id:'tex-abzac',        file:'s12', node:'Абзац',                         section:'12. ТЕКСТОВЕДЕНИЕ',    title:'Абзац',                     icon:'fa-paragraph',     xp:40 },
];

// Быстрый доступ по id и по названию узла дерева.
const LESSON_BY_ID   = Object.fromEntries(LESSON_INDEX.map(l => [l.id, l]));
const LESSON_BY_NODE = Object.fromEntries(LESSON_INDEX.map(l => [l.node, l]));

/**
 * Урок для узла дерева. Подпись узла в дереве не всегда уникальна
 * («Однокоренные слова», «Прямая речь» встречаются дважды), поэтому у
 * таких уроков в индексе указан ещё и level — и он обязан совпасть.
 */
function lessonForNode(node) {
  const l = LESSON_BY_NODE[node.label];
  if (!l) return null;
  if (l.level !== undefined && node.level !== l.level) return null;
  return l;
}

// Разделы в порядке дерева — для навигации «дальше» и группировки.
const LESSON_SECTIONS = LESSON_INDEX.reduce((acc, l) => {
  (acc[l.section] = acc[l.section] || []).push(l);
  return acc;
}, {});

/**
 * Подгружает файл раздела с содержанием уроков. Один и тот же файл
 * повторно не запрашивается: браузер кеширует, а мы держим промисы.
 */
const _lessonFilePromises = {};
function loadLessonFile(file) {
  if (_lessonFilePromises[file]) return _lessonFilePromises[file];
  _lessonFilePromises[file] = new Promise((resolve, reject) => {
    const s = document.createElement('script');
    s.src = `lessons/${file}.js`;
    s.onload = () => resolve();
    s.onerror = () => reject(new Error('Не удалось загрузить ' + file));
    document.head.appendChild(s);
  });
  return _lessonFilePromises[file];
}

/** Возвращает полный урок по id, подгрузив нужный файл раздела. */
async function loadLesson(id) {
  const meta = LESSON_BY_ID[id];
  if (!meta) return null;
  await loadLessonFile(meta.file);
  const data = (window.LESSONS || {})[id];
  return data ? { ...meta, ...data } : null;
}

/** Следующий урок по порядку дерева — для кнопки «Дальше» в конце. */
function nextLesson(id) {
  const i = LESSON_INDEX.findIndex(l => l.id === id);
  return (i >= 0 && i + 1 < LESSON_INDEX.length) ? LESSON_INDEX[i + 1] : null;
}

/** Порядковый номер урока внутри своего раздела — для подписи «3 из 12». */
function lessonPosition(id) {
  const meta = LESSON_BY_ID[id];
  if (!meta) return null;
  const group = LESSON_SECTIONS[meta.section];
  return { index: group.indexOf(meta) + 1, total: group.length };
}
