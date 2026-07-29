// RootRy API Client — используется на всех страницах
const API = {
    base: '',

    token() { return localStorage.getItem('token'); },
    user() { try { return JSON.parse(localStorage.getItem('currentUser')); } catch { return null; } },

    headers() {
        const h = { 'Content-Type': 'application/json' };
        const t = this.token();
        if (t) h['Authorization'] = 'Bearer ' + t;
        return h;
    },

    async get(path) {
        const res = await fetch(this.base + path, { headers: this.headers() });
        return this._handle(res);
    },

    async post(path, body) {
        const res = await fetch(this.base + path, {
            method: 'POST', headers: this.headers(), body: JSON.stringify(body)
        });
        return this._handle(res);
    },

    async put(path, body) {
        const res = await fetch(this.base + path, {
            method: 'PUT', headers: this.headers(), body: JSON.stringify(body)
        });
        return this._handle(res);
    },

    async _handle(res) {
        const data = await res.json().catch(() => ({}));
        if (res.status === 401) {
            localStorage.removeItem('token');
            localStorage.removeItem('currentUser');
            window.location.href = '/index.html';
            return { ok: false, status: 401, data };
        }
        return { ok: res.ok, status: res.status, data };
    },

    logout() {
        localStorage.removeItem('token');
        localStorage.removeItem('currentUser');
        window.location.href = '/index.html';
    },

    // Свежие данные пользователя.
    //
    // Раньше каждая страница дёргала /api/me дважды: один раз из initHeader,
    // второй — из своего загрузчика. Теперь параллельные вызовы разделяют
    // один запрос, а результат недолго живёт в памяти, поэтому переход между
    // вкладками не начинается с двух одинаковых обращений к серверу.
    _pending: null,
    _freshAt: 0,

    async refreshUser({ force = false } = {}) {
        const age = Date.now() - this._freshAt;
        if (!force && this._pending) return this._pending;
        if (!force && age < 1500) return this.user();

        this._pending = (async () => {
            const r = await this.get('/api/me');
            if (r.ok) {
                localStorage.setItem('currentUser', JSON.stringify(r.data));
                this._freshAt = Date.now();
                return r.data;
            }
            return this.user();
        })();

        try {
            return await this._pending;
        } finally {
            this._pending = null;
        }
    }
};

// ── Аватарки ────────────────────────────────────────────────────────
// Каталог общий для профиля и шапки: иначе выбранная аватарка
// показывалась бы в одном месте и не показывалась в другом.
const ALL_AVATARS = {
    common: [
        { emoji:'🐱', label:'Кот',     img:'https://api.dicebear.com/7.x/bottts/svg?seed=cat&backgroundColor=b6e3f4' },
        { emoji:'🐶', label:'Пёс',     img:'https://api.dicebear.com/7.x/bottts/svg?seed=dog&backgroundColor=ffd5dc' },
        { emoji:'🦊', label:'Лис',     img:'https://api.dicebear.com/7.x/bottts/svg?seed=fox&backgroundColor=c0aede' },
        { emoji:'🐼', label:'Панда',   img:'https://api.dicebear.com/7.x/bottts/svg?seed=panda&backgroundColor=d1f4e0' },
        { emoji:'🐻', label:'Медведь', img:'https://api.dicebear.com/7.x/bottts/svg?seed=bear&backgroundColor=ffd5dc' },
        { emoji:'🐸', label:'Лягушка', img:'https://api.dicebear.com/7.x/bottts/svg?seed=frog&backgroundColor=d1f4e0' },
        { emoji:'🦁', label:'Лев',     img:'https://api.dicebear.com/7.x/bottts/svg?seed=lion&backgroundColor=fde68a' },
        { emoji:'🐯', label:'Тигр',    img:'https://api.dicebear.com/7.x/bottts/svg?seed=tiger&backgroundColor=fed7aa' },
    ],
    rare: [
        { emoji:'🦄', label:'Единорог', img:'https://api.dicebear.com/7.x/bottts/svg?seed=unicorn&backgroundColor=bfdbfe' },
        { emoji:'🐉', label:'Дракон',   img:'https://api.dicebear.com/7.x/bottts/svg?seed=dragon&backgroundColor=93c5fd' },
        { emoji:'🦋', label:'Бабочка',  img:'https://api.dicebear.com/7.x/bottts/svg?seed=butterfly&backgroundColor=c4b5fd' },
        { emoji:'🦚', label:'Павлин',   img:'https://api.dicebear.com/7.x/bottts/svg?seed=peacock&backgroundColor=a5f3fc' },
        { emoji:'🦜', label:'Попугай',  img:'https://api.dicebear.com/7.x/bottts/svg?seed=parrot&backgroundColor=bbf7d0' },
        { emoji:'🦩', label:'Фламинго', img:'https://api.dicebear.com/7.x/bottts/svg?seed=flamingo&backgroundColor=fecdd3' },
        { emoji:'🐬', label:'Дельфин',  img:'https://api.dicebear.com/7.x/bottts/svg?seed=dolphin&backgroundColor=bae6fd' },
    ],
    epic: [
        { emoji:'🧙', label:'Маг',    img:'https://api.dicebear.com/7.x/bottts/svg?seed=wizard&backgroundColor=ddd6fe' },
        { emoji:'🧛', label:'Вампир', img:'https://api.dicebear.com/7.x/bottts/svg?seed=vampire&backgroundColor=fecdd3' },
        { emoji:'🦸', label:'Герой',  img:'https://api.dicebear.com/7.x/bottts/svg?seed=hero&backgroundColor=d9f99d' },
        { emoji:'🧝', label:'Эльф',   img:'https://api.dicebear.com/7.x/bottts/svg?seed=elf&backgroundColor=a7f3d0' },
        { emoji:'🧜', label:'Русалка',img:'https://api.dicebear.com/7.x/bottts/svg?seed=mermaid&backgroundColor=7dd3fc' },
    ],
    legendary: [
        { emoji:'👑', label:'Корона', img:'https://api.dicebear.com/7.x/bottts/svg?seed=crown&backgroundColor=fef08a' },
        { emoji:'🌟', label:'Звезда', img:'https://api.dicebear.com/7.x/bottts/svg?seed=star&backgroundColor=fde68a' },
        { emoji:'💫', label:'Комета', img:'https://api.dicebear.com/7.x/bottts/svg?seed=comet&backgroundColor=fef9c3' },
    ],
    mythic: [
        { emoji:'🌈', label:'Радуга', img:'https://api.dicebear.com/7.x/bottts/svg?seed=rainbow&backgroundColor=fbcfe8' },
    ],
};
const RARITY_LABELS = { common:'Обычная', rare:'Редкая', epic:'Эпическая', legendary:'Легендарная', mythic:'Мифическая' };
const RARITY_ORDER = ['common','rare','epic','legendary','mythic'];

// Картинка активной аватарки пользователя.
// Если аватарка не выбрана или неизвестна — рисуем робота по логину.
function avatarSrc(user) {
    const found = user && findAvatarData(user.active_avatar);
    if (found) return found.img;
    const seed = (user && user.username) || "guest";
    return 'https://api.dicebear.com/7.x/bottts/svg?seed=' + encodeURIComponent(seed);
}

function findAvatarData(emoji) {
    for (const rarity of RARITY_ORDER) {
        const found = (ALL_AVATARS[rarity] || []).find(a => a.emoji === emoji);
        if (found) return found;
    }
    return null;
}

function requireAuth() {
    if (!API.token()) {
        window.location.href = '/index.html';
        return false;
    }
    return true;
}

// Safe text helper — prevents XSS when setting element content from API data
function safeText(el, text) {
    if (el) el.textContent = text ?? '';
}

async function initHeader() {
    if (!requireAuth()) return;
    let user = API.user();
    if (!user) return;

    const applyUser = (u) => {
        safeText(document.getElementById('headerUsername'), u.nickname || u.username);
        safeText(document.getElementById('userBalance'), u.balance ?? 0);
        const av = document.getElementById('headerAvatar');
        if (av) av.src = avatarSrc(u);
    };

    applyUser(user);
    addSoundToggle();

    // Обновляем в фоне полными данными с сервера.
    API.refreshUser().then(u => { if (u) applyUser(u); });
}


// Кнопка «звук вкл/выкл» появляется в шапке на всех страницах.
function addSoundToggle() {
    const right = document.querySelector('.header-right');
    if (!right || right.querySelector('.sound-toggle')) return;

    const btn = document.createElement('button');
    btn.className = 'sound-toggle';
    btn.textContent = Sound.muted ? '🔇' : '🔊';
    btn.title = Sound.muted ? 'Включить звук' : 'Выключить звук';
    btn.onclick = () => Sound.toggle();
    right.appendChild(btn);
}

function showToast(msg, type = 'info') {
    let toast = document.getElementById('globalToast');
    if (!toast) {
        toast = document.createElement('div');
        toast.id = 'globalToast';
        toast.style.cssText = 'position:fixed;top:85px;left:50%;transform:translateX(-50%) translateY(-10px);padding:11px 26px;border-radius:20px;font-family:Montserrat,sans-serif;font-weight:600;font-size:13px;z-index:9999;opacity:0;transition:0.3s;box-shadow:0 6px 20px rgba(0,0,0,0.15);pointer-events:none;white-space:nowrap;max-width:90vw;text-align:center;';
        document.body.appendChild(toast);
    }
    if (type === 'success') Sound.correct();
    else if (type === 'error') Sound.error();

    const colors = { success: '#2ecc71', error: '#e74c3c', info: '#023e50', warn: '#f39c12' };
    toast.style.background = colors[type] || colors.info;
    toast.style.color = '#fff';
    toast.textContent = msg; // textContent = safe
    toast.style.opacity = '1';
    toast.style.transform = 'translateX(-50%) translateY(0)';
    clearTimeout(toast._t);
    toast._t = setTimeout(() => {
        toast.style.opacity = '0';
        toast.style.transform = 'translateX(-50%) translateY(-10px)';
    }, 2800);
}
// Show reward notification in top-right corner
function showRewardNotification(lines) {
    // Remove existing
    const old = document.getElementById('rewardNotif');
    if (old) old.remove();

    const notif = document.createElement('div');
    notif.id = 'rewardNotif';
    notif.style.cssText = [
        'position:fixed',
        'top:90px',
        'right:20px',
        'background:linear-gradient(135deg,#023e50,#0099cc)',
        'color:#fff',
        'padding:14px 20px',
        'border-radius:18px',
        'font-family:Montserrat,sans-serif',
        'font-size:14px',
        'font-weight:600',
        'z-index:99999',
        'box-shadow:0 8px 30px rgba(0,153,204,0.35)',
        'opacity:0',
        'transform:translateX(40px)',
        'transition:0.35s cubic-bezier(.4,0,.2,1)',
        'min-width:200px',
        'pointer-events:none',
    ].join(';');

    notif.innerHTML = lines.map(l => `<div style="margin:3px 0">${l}</div>`).join('');
    document.body.appendChild(notif);

    requestAnimationFrame(() => {
        notif.style.opacity = '1';
        notif.style.transform = 'translateX(0)';
    });

    setTimeout(() => {
        notif.style.opacity = '0';
        notif.style.transform = 'translateX(40px)';
        setTimeout(() => notif.remove(), 400);
    }, 3500);
}

// Submit game result to backend — call this at the end of every game
async function submitGameResult(gameType, score) {
    try {
        const r = await API.post('/api/game/submit', { game_type: gameType, score: score });
        if (r.ok) {
            // Монеты летят в кошелёк, счётчик докручивается.
            if (r.data.new_balance !== undefined) {
                const earned = r.data.coins_earned || 0;
                if (earned > 0) rewardCoins(earned, r.data.new_balance);
                else animateBalance(r.data.new_balance);
            }
            // Update cached user including quest-related fields
            const u = API.user();
            if (u && r.data.new_balance !== undefined) {
                u.balance = r.data.new_balance;
                u.xp = r.data.new_xp;
                // Keep games_won_today and daily_tasks_date in sync so shop shows correct quest progress
                if (r.data.games_won_today !== undefined) {
                    u.games_won_today = r.data.games_won_today;
                    const today = new Date().toISOString().slice(0, 10);
                    u.daily_tasks_date = today;
                }
                if (r.data.quest_bonus_earned) {
                    u.daily_tasks_done = 1;
                }
                localStorage.setItem('currentUser', JSON.stringify(u));
            }

            const isWin = score > 0;
            const firstWin = r.data.first_win;
            const coins = r.data.coins_earned || 0;
            const xp = r.data.xp_earned || 0;
            const questBonus = r.data.quest_bonus_earned;
            const gamesWon = r.data.games_won_today || 0;

            if (isWin) {
                const lines = [];
                if (firstWin) {
                    lines.push(`🪙 +50 монет — первая победа в этой игре!`);
                } else {
                    lines.push('✅ Игра пройдена');
                    lines.push('💡 50 монет уже получены ранее за эту игру');
                }
                if (xp > 0) lines.push(`⚡ +${xp} XP`);
                if (questBonus) {
                    lines.push('');
                    lines.push(`🎯 Дейлик выполнен! +50 монет`);
                } else if (firstWin && gamesWon < 5) {
                    lines.push(`📊 Побед в разных играх: ${gamesWon}/5`);
                }
                if (r.data.badge_earned) lines.push(`🏅 Новый значок: ${r.data.badge_earned}`);
                if (lines.length) showRewardNotification(lines);
                if (firstWin) Sound.win();
            }
        }
        return r;
    } catch(e) { return { ok: false }; }
}

// ── Звук ────────────────────────────────────────────────────────────
// Короткие сигналы синтезируются через WebAudio: никаких файлов,
// лишних запросов и задержек на загрузку.
// Контекст создаётся при первом клике — браузеры не дают включить
// звук до действия пользователя.
const Sound = {
    ctx: null,
    muted: localStorage.getItem('soundMuted') === '1',

    ensure() {
        if (this.muted) return null;
        if (!this.ctx) {
            const Ctx = window.AudioContext || window.webkitAudioContext;
            if (!Ctx) return null;
            this.ctx = new Ctx();
        }
        if (this.ctx.state === 'suspended') this.ctx.resume();
        return this.ctx;
    },

    toggle() {
        this.muted = !this.muted;
        localStorage.setItem('soundMuted', this.muted ? '1' : '0');
        if (!this.muted) this.click();
        document.querySelectorAll('.sound-toggle').forEach(b => {
            b.textContent = this.muted ? '🔇' : '🔊';
            b.title = this.muted ? 'Включить звук' : 'Выключить звук';
        });
        return this.muted;
    },

    // Один тон. type — форма волны, freq — частота, dur — длительность.
    tone(freq, dur, { type = 'sine', gain = 0.06, slideTo = null, delay = 0 } = {}) {
        const ctx = this.ensure();
        if (!ctx) return;
        const t0 = ctx.currentTime + delay;

        const osc = ctx.createOscillator();
        const vol = ctx.createGain();
        osc.type = type;
        osc.frequency.setValueAtTime(freq, t0);
        if (slideTo) osc.frequency.exponentialRampToValueAtTime(slideTo, t0 + dur);

        // Мягкая атака и затухание, иначе на концах слышны щелчки.
        vol.gain.setValueAtTime(0.0001, t0);
        vol.gain.exponentialRampToValueAtTime(gain, t0 + 0.012);
        vol.gain.exponentialRampToValueAtTime(0.0001, t0 + dur);

        osc.connect(vol);
        vol.connect(ctx.destination);
        osc.start(t0);
        osc.stop(t0 + dur + 0.02);
    },

    click()  { this.tone(520, 0.05, { type: 'triangle', gain: 0.035 }); },
    hover()  { this.tone(700, 0.03, { type: 'sine', gain: 0.015 }); },
    coin()   { this.tone(880, 0.09, { type: 'square', gain: 0.03 });
               this.tone(1320, 0.12, { type: 'square', gain: 0.025, delay: 0.06 }); },
    correct(){ this.tone(660, 0.1, { type: 'sine' });
               this.tone(990, 0.16, { type: 'sine', delay: 0.08 }); },
    wrong()  { this.tone(220, 0.22, { type: 'sawtooth', gain: 0.04, slideTo: 120 }); },
    error()  { this.tone(180, 0.25, { type: 'square', gain: 0.035, slideTo: 110 }); },
    win()    { [523, 659, 784, 1047].forEach((f, i) =>
                 this.tone(f, 0.22, { type: 'triangle', gain: 0.05, delay: i * 0.09 })); },
    open()   { this.tone(300, 0.5, { type: 'sine', gain: 0.05, slideTo: 1200 }); },
    levelUp(){ [392, 523, 659, 784, 1047].forEach((f, i) =>
                 this.tone(f, 0.3, { type: 'sine', gain: 0.055, delay: i * 0.1 })); },
};

// Клик по любой кнопке или ссылке-вкладке звучит одинаково.
document.addEventListener('click', e => {
    const el = e.target.closest('button, .tab-item, .lb-tab, .case-card');
    if (el && !el.disabled && !el.classList.contains('sound-toggle')) Sound.click();
}, true);

// ── Монеты ──────────────────────────────────────────────────────────
// Счётчик в шапке докручивается до нового значения, а от места события
// к кошельку летят монетки.
function animateBalance(to, from) {
    const el = document.getElementById('userBalance');
    if (!el) return;

    const start = from != null ? from : parseInt(el.textContent.replace(/\s/g, ''), 10) || 0;
    const target = Number(to) || 0;
    if (start === target) { el.textContent = target; return; }

    const duration = 700;
    const t0 = performance.now();

    function step(now) {
        const p = Math.min(1, (now - t0) / duration);
        // Замедление к концу, чтобы последние цифры читались.
        const eased = 1 - Math.pow(1 - p, 3);
        el.textContent = Math.round(start + (target - start) * eased);
        if (p < 1) requestAnimationFrame(step);
        else {
            el.textContent = target;
            el.classList.remove('balance-bump');
            void el.offsetWidth;
            el.classList.add('balance-bump');
        }
    }
    requestAnimationFrame(step);
}

// Монетки, летящие к кошельку. origin — элемент, от которого лететь.
function flyCoins(origin, count) {
    const target = document.getElementById('userBalance');
    if (!target) return;

    const to = target.getBoundingClientRect();
    const from = origin && origin.getBoundingClientRect
        ? origin.getBoundingClientRect()
        : { left: innerWidth / 2, top: innerHeight / 2, width: 0, height: 0 };

    const n = Math.min(Math.max(count || 1, 1), 12);
    for (let i = 0; i < n; i++) {
        const coin = document.createElement('div');
        coin.className = 'flying-coin';
        coin.textContent = '🪙';
        coin.style.left = (from.left + from.width / 2) + 'px';
        coin.style.top = (from.top + from.height / 2) + 'px';
        document.body.appendChild(coin);

        // Небольшой разлёт, чтобы монеты не летели одной линией.
        const spreadX = (Math.random() - 0.5) * 90;
        const spreadY = (Math.random() - 0.5) * 60;

        requestAnimationFrame(() => {
            coin.style.transition = 'transform 0.75s cubic-bezier(0.4,0,0.25,1), opacity 0.75s';
            coin.style.transitionDelay = (i * 0.05) + 's';
            coin.style.transform =
                `translate(${to.left + to.width / 2 - from.left - from.width / 2 + spreadX}px,
                           ${to.top + to.height / 2 - from.top - from.height / 2 + spreadY}px) scale(0.5)`;
            coin.style.opacity = '0';
        });

        setTimeout(() => coin.remove(), 1100 + i * 50);
    }
    Sound.coin();
}

// Награда монетами: и полёт, и докрутка счётчика, и звук.
function rewardCoins(amount, newBalance, origin) {
    if (!amount) return;
    flyCoins(origin, Math.ceil(amount / 25));
    if (newBalance != null) animateBalance(newBalance);
}

// Стили для монет и подскока счётчика — чтобы каждая страница
// не описывала их у себя.
(function injectCoinStyles() {
    const css = document.createElement('style');
    css.textContent = `
        .flying-coin {
            position: fixed; z-index: 99998; font-size: 22px;
            pointer-events: none; will-change: transform, opacity;
        }
        .balance-bump { animation: balanceBump 0.45s cubic-bezier(0.34,1.56,0.64,1); }
        @keyframes balanceBump {
            0%   { transform: scale(1); }
            45%  { transform: scale(1.35); color: #fcd34d; }
            100% { transform: scale(1); }
        }
        .sound-toggle {
            border: none; background: rgba(255,255,255,0.18); color: #fff;
            width: 34px; height: 34px; border-radius: 50%; cursor: pointer;
            font-size: 15px; line-height: 1; transition: 0.2s;
        }
        .sound-toggle:hover { background: rgba(255,255,255,0.32); }
        @media (prefers-reduced-motion: reduce) {
            .flying-coin { display: none; }
            .balance-bump { animation: none; }
        }
    `;
    document.head.appendChild(css);
})();
