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

    // Fix: refreshUser replaces full user object, not a partial spread
    async refreshUser() {
        const r = await this.get('/api/me');
        if (r.ok) {
            localStorage.setItem('currentUser', JSON.stringify(r.data));
            return r.data;
        }
        return this.user();
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
    // Refresh in background with full server data
    API.refreshUser().then(u => { if (u) applyUser(u); });
}

function showToast(msg, type = 'info') {
    let toast = document.getElementById('globalToast');
    if (!toast) {
        toast = document.createElement('div');
        toast.id = 'globalToast';
        toast.style.cssText = 'position:fixed;top:85px;left:50%;transform:translateX(-50%) translateY(-10px);padding:11px 26px;border-radius:20px;font-family:Montserrat,sans-serif;font-weight:600;font-size:13px;z-index:9999;opacity:0;transition:0.3s;box-shadow:0 6px 20px rgba(0,0,0,0.15);pointer-events:none;white-space:nowrap;max-width:90vw;text-align:center;';
        document.body.appendChild(toast);
    }
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
            // Update balance in header
            if (r.data.new_balance !== undefined) {
                const balEl = document.getElementById('userBalance');
                if (balEl) balEl.textContent = r.data.new_balance;
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
            }
        }
        return r;
    } catch(e) { return { ok: false }; }
}
