/* ═══════════════════════════════════════════════════════════════════
   practice.js — общая механика заданий.

   Задания одного и того же формата решаются в двух местах: в уроке и в
   работе над ошибками. Проверка ответа обязана быть общей — если бы
   «верно» определялось в двух файлах по отдельности, они рано или поздно
   разошлись бы, и одно и то же задание засчитывалось бы по-разному в
   зависимости от того, откуда ученик его открыл.

   Здесь два независимых куска:
     practiceIsCorrect — чистая проверка ответа, без DOM;
     PracticeDrill     — виджет прорешивания для тетради ошибок.
   ═══════════════════════════════════════════════════════════════════ */

const PRACTICE_KINDS = {
    choice: 'Выбери ответ',
    fill:   'Вставь пропущенное',
    tf:     'Верно или неверно',
    multi:  'Выбери все верные',
    match:  'Сопоставь пары',
    order:  'Расставь по порядку',
};

/**
 * Проверить ответ на задание.
 *
 * @param {object} q задание из lessons/s*.js
 * @param {object} a ответ ученика: { pick, matchDone, matchErrors, order }
 * @returns {boolean}
 */
function practiceIsCorrect(q, a) {
    a = a || {};
    switch (q.type) {
        case 'tf':
            return a.pick === q.correct;
        case 'multi': {
            const got = [...(a.pick || [])].sort().join(',');
            const want = [...(q.correct || [])].sort().join(',');
            return got === want && got !== '';
        }
        // Пары в «сопоставь» в итоге сходятся всегда: неверную ученик просто
        // пробует заново. Поэтому засчитывается только сопоставление,
        // собранное без единой ошибки.
        case 'match':
            return (a.matchDone || []).length === (q.pairs || []).length
                && !a.matchErrors;
        case 'order':
            return (a.order || []).join(',') === (q.correct || []).join(',');
        default:
            return a.pick === q.correct;
    }
}

/* ═══════════════════════════════════════════════════════════════════
   PracticeDrill — прорешивание списка заданий подряд.

   Используется в тетради ошибок. От урока отличается намеренно: нет
   жизней, нет комбо, нет «провалил тему». Работа над ошибками не должна
   наказывать — задание, которое не даётся, просто вернётся завтра.
   ═══════════════════════════════════════════════════════════════════ */
const PracticeDrill = {
    _s: null,

    /**
     * @param {HTMLElement} root куда рисовать
     * @param {Array} items [{ topicId, itemId, title, q }]
     * @param {object} opts { onAnswer(item, correct, timeMs), onDone(stats) }
     */
    mount(root, items, opts = {}) {
        this._s = {
            root, items, opts,
            idx: 0, correct: 0,
            answered: false,
            startedAt: 0,
            pick: null, order: [], matchLeft: null, matchDone: [], matchErrors: 0,
        };
        this._renderQuestion();
    },

    _esc(v) {
        return String(v ?? '').replace(/[&<>"']/g, c =>
            ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
    },

    /* Пропуск в тексте задания обозначен подчёркиваниями. */
    _formatGap(text) {
        return this._esc(text).replace(/_{2,}/g, '<span class="pd-gap">_____</span>');
    },

    _renderQuestion() {
        const s = this._s;
        if (s.idx >= s.items.length) return this._renderDone();

        const item = s.items[s.idx];
        const q = item.q;
        s.answered = false;
        s.pick = null; s.order = []; s.matchLeft = null; s.matchDone = []; s.matchErrors = 0;
        s.startedAt = Date.now();

        s.root.innerHTML = `
            <div class="pd-head">
                <span class="pd-kind">${this._esc(PRACTICE_KINDS[q.type] || 'Задание')}</span>
                <span class="pd-count">${s.idx + 1} из ${s.items.length}</span>
            </div>
            <div class="pd-topic">${this._esc(item.title || item.topicId)}</div>
            <div class="pd-q">${this._formatGap(q.q)}</div>
            <div class="pd-body" id="pdBody"></div>
            <div class="pd-feedback" id="pdFeedback" hidden></div>
            <button class="pd-btn" id="pdBtn" disabled>Проверить</button>`;

        const body = s.root.querySelector('#pdBody');
        ({
            choice: this._renderOptions, fill: this._renderOptions,
            tf: this._renderTF, multi: this._renderMulti,
            match: this._renderMatch, order: this._renderOrder,
        }[q.type] || this._renderOptions).call(this, body, q);

        s.root.querySelector('#pdBtn').onclick = () => this._check();
    },

    _btnEnabled(on) {
        const b = this._s.root.querySelector('#pdBtn');
        if (b) b.disabled = !on;
    },

    _renderOptions(body, q) {
        body.innerHTML = (q.options || [])
            .map((o, i) => `<button class="pd-opt" data-i="${i}">${this._esc(o)}</button>`).join('');
        body.querySelectorAll('.pd-opt').forEach(b => {
            b.onclick = () => {
                if (this._s.answered) return;
                body.querySelectorAll('.pd-opt').forEach(x => x.classList.remove('sel'));
                b.classList.add('sel');
                this._s.pick = +b.dataset.i;
                this._btnEnabled(true);
            };
        });
    },

    _renderTF(body, q) {
        body.innerHTML =
            '<button class="pd-opt" data-v="1">Верно</button>' +
            '<button class="pd-opt" data-v="0">Неверно</button>';
        body.querySelectorAll('.pd-opt').forEach(b => {
            b.onclick = () => {
                if (this._s.answered) return;
                body.querySelectorAll('.pd-opt').forEach(x => x.classList.remove('sel'));
                b.classList.add('sel');
                this._s.pick = b.dataset.v === '1';
                this._btnEnabled(true);
            };
        });
    },

    _renderMulti(body, q) {
        this._s.pick = [];
        body.innerHTML = (q.options || [])
            .map((o, i) => `<button class="pd-opt" data-i="${i}">${this._esc(o)}</button>`).join('');
        body.querySelectorAll('.pd-opt').forEach(b => {
            b.onclick = () => {
                if (this._s.answered) return;
                const i = +b.dataset.i;
                const at = this._s.pick.indexOf(i);
                if (at >= 0) { this._s.pick.splice(at, 1); b.classList.remove('sel'); }
                else { this._s.pick.push(i); b.classList.add('sel'); }
                this._btnEnabled(this._s.pick.length > 0);
            };
        });
    },

    _renderOrder(body, q) {
        const items = q.items || [];
        body.innerHTML =
            `<div class="pd-slots" id="pdSlots"></div>
             <div class="pd-pool">${items.map((t, i) =>
                `<button class="pd-tile" data-i="${i}">${this._esc(t)}</button>`).join('')}</div>`;
        const slots = body.querySelector('#pdSlots');
        body.querySelectorAll('.pd-tile').forEach(b => {
            b.onclick = () => {
                if (this._s.answered || b.classList.contains('used')) return;
                b.classList.add('used');
                this._s.order.push(+b.dataset.i);
                slots.insertAdjacentHTML('beforeend',
                    `<span class="pd-slot">${this._esc(items[+b.dataset.i])}</span>`);
                this._btnEnabled(this._s.order.length === items.length);
            };
        });
    },

    _renderMatch(body, q) {
        const pairs = q.pairs || [];
        // Правую колонку перемешиваем, иначе сопоставление решается взглядом.
        const right = pairs.map((p, i) => ({ text: p[1], i }));
        for (let i = right.length - 1; i > 0; i--) {
            const j = Math.floor(Math.random() * (i + 1));
            [right[i], right[j]] = [right[j], right[i]];
        }
        body.innerHTML =
            `<div class="pd-match">
                <div class="pd-col">${pairs.map((p, i) =>
                    `<button class="pd-tile" data-side="l" data-i="${i}">${this._esc(p[0])}</button>`).join('')}</div>
                <div class="pd-col">${right.map(r =>
                    `<button class="pd-tile" data-side="r" data-i="${r.i}">${this._esc(r.text)}</button>`).join('')}</div>
             </div>`;

        body.querySelectorAll('.pd-tile').forEach(b => {
            b.onclick = () => {
                const s = this._s;
                if (s.answered || b.classList.contains('done')) return;
                if (b.dataset.side === 'l') {
                    body.querySelectorAll('[data-side="l"]').forEach(x => x.classList.remove('sel'));
                    b.classList.add('sel');
                    s.matchLeft = +b.dataset.i;
                    return;
                }
                if (s.matchLeft === null) return;
                const left = body.querySelector(`[data-side="l"][data-i="${s.matchLeft}"]`);
                if (+b.dataset.i === s.matchLeft) {
                    b.classList.add('done'); left.classList.add('done');
                    left.classList.remove('sel');
                    s.matchDone.push(s.matchLeft);
                    s.matchLeft = null;
                    if (s.matchDone.length === pairs.length) this._btnEnabled(true);
                } else {
                    s.matchErrors++;
                    b.classList.add('miss');
                    setTimeout(() => b.classList.remove('miss'), 400);
                }
            };
        });
    },

    _check() {
        const s = this._s;
        if (s.answered) {
            s.idx++;
            return this._renderQuestion();
        }
        s.answered = true;

        const item = s.items[s.idx];
        const ok = practiceIsCorrect(item.q, s);
        if (ok) s.correct++;

        this._markAnswers(item.q, ok);

        const fb = s.root.querySelector('#pdFeedback');
        fb.hidden = false;
        fb.className = 'pd-feedback ' + (ok ? 'good' : 'bad');
        fb.innerHTML =
            `<div class="pd-verdict">${ok ? '✓ Верно' : '✗ Мимо'}</div>` +
            (item.q.why ? `<div class="pd-why">${this._esc(item.q.why)}</div>` : '');

        const btn = s.root.querySelector('#pdBtn');
        btn.disabled = false;
        btn.textContent = s.idx === s.items.length - 1 ? 'Завершить' : 'Дальше';

        if (s.opts.onAnswer) {
            s.opts.onAnswer(item, ok, s.startedAt ? Date.now() - s.startedAt : 0);
        }
    },

    _markAnswers(q, ok) {
        const body = this._s.root.querySelector('#pdBody');
        if (!body) return;
        body.querySelectorAll('.pd-tile').forEach(t => t.disabled = true);
        body.querySelectorAll('.pd-opt').forEach(b => {
            b.disabled = true;
            const i = +b.dataset.i;
            const right = q.type === 'tf'  ? (b.dataset.v === '1') === q.correct
                        : q.type === 'multi' ? (q.correct || []).includes(i)
                        : i === q.correct;
            if (right) b.classList.add('ok');
            else if (b.classList.contains('sel')) b.classList.add('bad');
            b.classList.remove('sel');
        });
        if (q.type === 'order' && !ok) {
            const slots = body.querySelector('#pdSlots');
            if (slots) {
                slots.querySelectorAll('.pd-slot').forEach((el, pos) => {
                    el.classList.add(this._s.order[pos] === q.correct[pos] ? 'ok' : 'bad');
                });
            }
        }
    },

    _renderDone() {
        const s = this._s;
        s.root.innerHTML = `
            <div class="pd-done">
                <div class="pd-done-icon">${s.correct === s.items.length ? '🎯' : '💪'}</div>
                <div class="pd-done-title">Разбор окончен</div>
                <div class="pd-done-score">${s.correct} из ${s.items.length} верно</div>
                <div class="pd-done-note">Задание уходит из тетради после двух верных
                    ответов в разные дни — то, что не далось, вернётся завтра.</div>
            </div>`;
        if (s.opts.onDone) s.opts.onDone({ correct: s.correct, total: s.items.length });
    },
};
