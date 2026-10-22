/* ═══════════════════════════════════════════════════════════════════
   Проверка механики заданий.

   Запуск:  node static/practice.test.js

   Проверяется чистая логика practiceIsCorrect и лестницы подсказок —
   без браузера и без сервера. Именно эта функция решает, засчитан ли
   ответ, и решает одинаково в уроке и в тетради ошибок, поэтому ошибка
   здесь стоит дороже любой другой в клиенте.
   ═══════════════════════════════════════════════════════════════════ */

const fs = require('fs');
const vm = require('vm');
const src = fs.readFileSync(require('path').join(__dirname, 'practice.js'), 'utf8');
const ctx = { window: {}, document: { addEventListener() {} } };
vm.createContext(ctx);
vm.runInContext(src, ctx);
const isCorrect = vm.runInContext('practiceIsCorrect', ctx);
const hints = vm.runInContext('PracticeHints', ctx);

let bad = 0;
const t = (name, got, want) => {
  const ok = got === want;
  if (!ok) { bad++; console.log(`  ПРОВАЛ ${name}: получено ${got}, ожидалось ${want}`); }
};

// ── write ──
const w = { type: 'write', q: 'пр_красный, пр_ехать', answers: ['е', 'и'] };
t('write верно',            isCorrect(w, { written: ['е', 'и'] }), true);
t('write регистр',          isCorrect(w, { written: ['Е', 'И'] }), true);
t('write пробелы',          isCorrect(w, { written: [' е ', 'и'] }), true);
t('write одна ошибка',      isCorrect(w, { written: ['и', 'и'] }), false);
t('write недозаполнено',    isCorrect(w, { written: ['е'] }), false);
t('write пусто',            isCorrect(w, {}), false);

// ── find ──
const f = { type: 'find', wrong: ['прекрастна', 'машына'] };
t('find верно',             isCorrect(f, { found: ['машына', 'прекрастна'] }), true);
t('find лишнее слово',      isCorrect(f, { found: ['машына', 'прекрастна', 'жизнь'] }), false);
t('find неполно',           isCorrect(f, { found: ['машына'] }), false);
t('find дубли не мешают',   isCorrect(f, { found: ['машына', 'машына', 'прекрастна'] }), true);

// ── explain ──
const e = { type: 'explain', keywords: [['-енн-', 'енн'], ['стык', 'основ'], ['исключен', 'стеклянн']] };
t('explain все три',        isCorrect(e, { text: 'Пишем енн, а ещё на стыке основы, и есть исключения' }), true);
t('explain две из трёх',    isCorrect(e, { text: 'суффикс енн и на стыке основы' }), false);
t('explain пусто',          isCorrect(e, { text: '' }), false);
t('explain без ключей',     isCorrect({ type: 'explain' }, { text: 'что-то' }), false);

// ── dictation ──
const dct = { type: 'dictation', answer: 'Не рассчитывай на удачу, приложи усилия.' };
t('dictation точно',        isCorrect(dct, { text: 'Не рассчитывай на удачу, приложи усилия.' }), true);
t('dictation регистр',      isCorrect(dct, { text: 'не рассчитывай на удачу, приложи усилия.' }), true);
t('dictation два пробела',  isCorrect(dct, { text: 'Не  рассчитывай на удачу,  приложи усилия.' }), true);
t('dictation без запятой',  isCorrect(dct, { text: 'Не рассчитывай на удачу приложи усилия.' }), false);
t('dictation с ошибкой',    isCorrect(dct, { text: 'Не расчитывай на удачу, приложи усилия.' }), false);

// ── старые типы не сломались ──
t('choice',                 isCorrect({ type: 'choice', correct: 2 }, { pick: 2 }), true);
t('tf',                     isCorrect({ type: 'tf', correct: false }, { pick: false }), true);
t('multi',                  isCorrect({ type: 'multi', correct: [0, 2] }, { pick: [2, 0] }), true);
t('order',                  isCorrect({ type: 'order', correct: [0, 1, 2] }, { order: [0, 1, 2] }), true);
t('match',                  isCorrect({ type: 'match', pairs: [[1, 2], [3, 4]] }, { matchDone: [0, 1], matchErrors: 0 }), true);

// ── лестница подсказок ──
const lesson = { rule: 'Общее правило урока' };
const full = hints.levels({ hint: 'намёк', why: 'разбор' }, lesson);
t('три ступени',            full.length, 3);
t('последняя раскрывает',   full[2].reveals, true);
const short = hints.levels({ why: 'разбор' }, lesson);
t('без намёка — две',       short.length, 2);
t('report разбор = 3',      hints.report(2, true), 3);
t('report две ступени = 2', hints.report(2, false), 2);
t('report одна = 1',        hints.report(1, false), 1);

console.log(bad ? `\n${bad} проверок провалено` : '\nвсе проверки прошли');
process.exit(bad ? 1 : 0);
