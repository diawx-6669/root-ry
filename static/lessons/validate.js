/* ═══════════════════════════════════════════════════════════════════
   Проверка целостности учебных данных.

   Запуск:  node static/lessons/validate.js

   Проверяет три вещи, которые легко разъезжаются при правках:
     1. каждый урок из индекса привязан к существующему узлу дерева,
        и узел этот в дереве ровно один;
     2. у каждого урока есть содержание, а у каждого задания —
        корректный правильный ответ;
     3. все ссылки «связано с темами» ведут на существующие уроки.

   Реестр тем на сервере (internal/topics/topics.go) сверяется отдельно
   тестом TestRegistryMatchesLessonIndex.
   ═══════════════════════════════════════════════════════════════════ */

const fs = require('fs');
const path = require('path');
const vm = require('vm');

const DIR = __dirname;
const STATIC = path.join(DIR, '..');

const errors = [];
const warns = [];
const fail = m => errors.push(m);
const warn = m => warns.push(m);

/* ── Песочница: index.js и файлы разделов — обычные скрипты ──────── */
const sandbox = { window: {}, document: { createElement: () => ({}), head: { appendChild() {} } } };
vm.createContext(sandbox);

function run(file) {
  vm.runInContext(fs.readFileSync(file, 'utf8'), sandbox, { filename: file });
}

run(path.join(DIR, 'index.js'));
// index.js объявляет LESSON_INDEX через const — это лексическая привязка
// контекста, а не свойство sandbox, поэтому читаем её выражением.
const INDEX = vm.runInContext('LESSON_INDEX', sandbox);

const files = [...new Set(INDEX.map(l => l.file))];
for (const f of files) run(path.join(DIR, f + '.js'));
const LESSONS = sandbox.window.LESSONS;

/* ── 1. Узлы дерева ─────────────────────────────────────────────── */
const treeSrc = fs.readFileSync(path.join(STATIC, 'tree.html'), 'utf8');
const lessonForNode = vm.runInContext('lessonForNode', sandbox);

// Узлы дерева: подпись + авторский уровень. Одной подписи может
// соответствовать несколько узлов на разных уровнях.
const nodes = [...treeSrc.matchAll(/\{label:"([^"]+)",\s*level:(\d)/g)]
  .map(m => ({ label: m[1], level: +m[2] }));

for (const l of INDEX) {
  const matched = nodes.filter(n => lessonForNode(n)?.id === l.id);
  if (matched.length === 0) fail(`[${l.id}] узла «${l.node}» нет в дереве`);
  else if (matched.length > 1)
    fail(`[${l.id}] узел «${l.node}» находится в дереве ${matched.length} раза — ` +
         `кнопка урока появится на всех, уточни level в индексе`);
}

/* ── 2. Содержание уроков ───────────────────────────────────────── */
const ids = new Set(INDEX.map(l => l.id));
if (ids.size !== INDEX.length) fail('в индексе есть повторяющиеся id');

for (const meta of INDEX) {
  const L = LESSONS[meta.id];
  if (!L) { fail(`[${meta.id}] нет содержания в ${meta.file}.js`); continue; }

  if (!L.intro) fail(`[${meta.id}] нет intro`);
  if (!L.rule) fail(`[${meta.id}] нет правила`);
  if (!Array.isArray(L.steps) || !L.steps.length) fail(`[${meta.id}] нет блоков теории`);
  if (!Array.isArray(L.practice) || L.practice.length < 4)
    fail(`[${meta.id}] меньше 4 заданий практики`);

  (L.steps || []).forEach((s, i) => {
    if (!s.h || !s.t) fail(`[${meta.id}] блок теории ${i + 1}: нет заголовка или текста`);
    if (s.table) {
      if (s.tableHead && s.table.some(r => r.length !== s.tableHead.length))
        fail(`[${meta.id}] блок ${i + 1}: число колонок не совпадает с заголовком`);
    }
  });

  (L.practice || []).forEach((q, i) => {
    const tag = `[${meta.id}] задание ${i + 1} (${q.type})`;
    if (!q.q) fail(`${tag}: нет текста вопроса`);
    if (!q.why) warn(`${tag}: нет объяснения ответа`);

    switch (q.type) {
      case 'choice':
      case 'fill':
        if (!Array.isArray(q.options) || q.options.length < 2) fail(`${tag}: нужно минимум 2 варианта`);
        else if (!Number.isInteger(q.correct) || q.correct < 0 || q.correct >= q.options.length)
          fail(`${tag}: correct=${q.correct} вне диапазона вариантов`);
        // Пропуск бывает двух видов: целое слово («___») и буква внутри
        // слова («предл_гает»). Плеер рисует оба, но хотя бы один нужен.
        if (q.type === 'fill' && !/_/.test(q.q)) warn(`${tag}: в тексте нет пропуска «_»`);
        break;
      case 'tf':
        if (typeof q.correct !== 'boolean') fail(`${tag}: correct должен быть true или false`);
        break;
      case 'multi':
        if (!Array.isArray(q.options) || !Array.isArray(q.correct) || !q.correct.length)
          fail(`${tag}: нужны options и непустой массив correct`);
        else if (q.correct.some(i2 => !Number.isInteger(i2) || i2 < 0 || i2 >= q.options.length))
          fail(`${tag}: индекс в correct вне диапазона`);
        else if (q.correct.length === q.options.length)
          warn(`${tag}: верны все варианты — задание не различает`);
        break;
      case 'match':
        if (!Array.isArray(q.pairs) || q.pairs.length < 2) fail(`${tag}: нужно минимум 2 пары`);
        else if (q.pairs.some(p => !Array.isArray(p) || p.length !== 2)) fail(`${tag}: пара должна быть [левое, правое]`);
        else if (new Set(q.pairs.map(p => p[1])).size !== q.pairs.length)
          fail(`${tag}: правые части повторяются — задание нерешаемо однозначно`);
        break;
      case 'order': {
        if (!Array.isArray(q.items) || !Array.isArray(q.correct)) { fail(`${tag}: нужны items и correct`); break; }
        if (q.items.length !== q.correct.length) { fail(`${tag}: длины items и correct не совпадают`); break; }
        const sorted = [...q.correct].sort((a, b) => a - b).join(',');
        const expect = q.items.map((_, i2) => i2).join(',');
        if (sorted !== expect) fail(`${tag}: correct должен быть перестановкой индексов items`);
        break;
      }
      default:
        fail(`${tag}: неизвестный тип задания`);
    }
  });

  (L.links || []).forEach(link => {
    if (!ids.has(link.id)) fail(`[${meta.id}] ссылка на несуществующий урок «${link.id}»`);
    if (link.id === meta.id) fail(`[${meta.id}] ссылка на самого себя`);
    if (!link.why) warn(`[${meta.id}] у ссылки на «${link.id}» нет пояснения`);
  });
  if (!L.links || L.links.length < 2) warn(`[${meta.id}] меньше 2 связей с другими темами`);
}

/* ── 3. Отчёт ───────────────────────────────────────────────────── */
const totalQ = INDEX.reduce((s, m) => s + (LESSONS[m.id]?.practice?.length || 0), 0);
const totalS = INDEX.reduce((s, m) => s + (LESSONS[m.id]?.steps?.length || 0), 0);
const totalL = INDEX.reduce((s, m) => s + (LESSONS[m.id]?.links?.length || 0), 0);

console.log(`Уроков:          ${INDEX.length}`);
console.log(`Блоков теории:   ${totalS}`);
console.log(`Заданий:         ${totalQ}`);
console.log(`Связей меж тем:  ${totalL}`);
console.log('');

if (warns.length) {
  console.log(`Предупреждения (${warns.length}):`);
  warns.forEach(w => console.log('  ! ' + w));
  console.log('');
}
if (errors.length) {
  console.log(`ОШИБКИ (${errors.length}):`);
  errors.forEach(e => console.log('  × ' + e));
  process.exit(1);
}
console.log('✓ Все проверки пройдены');
