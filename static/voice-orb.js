/* ═══════════════════════════════════════════════════════════════════
   Орб голосового ассистента — облако частиц с линиями связи.

   Идея взята из JARVIS (github.com/diawx-6669/jarvis): сфера частиц,
   между близкими протянуты линии, поведение меняется вместе с
   состоянием — покой, слушаю, думаю, говорю.

   Там это Three.js и WebGL. Здесь — обычный canvas 2D, и вот почему:
   платформа не тянет ни одной библиотеки, кроме d3 на дереве
   грамматики, а Three.js — это ещё шестьсот килобайт на страницу,
   которая должна открываться на школьном ноутбуке. Сфера из четырёхсот
   точек с проекцией и линиями рисуется в 2D без единой зависимости и
   держит шестьдесят кадров даже на встроенной графике.

   Что видно ученику:
     idle      — медленный дрейф, линий почти нет;
     listening — частицы расходятся и дышат, цвет синий;
     thinking  — быстрое вращение, цвет янтарный;
     speaking  — частицы стягиваются, линий много, цвет зелёный,
                 pulse() на каждом слове даёт толчок.
   ═══════════════════════════════════════════════════════════════ */

function createOrb(canvas) {
    if (!canvas || !canvas.getContext) {
        // Без canvas страница обязана работать: орб — украшение, а не
        // способ узнать, что происходит. Для этого есть подпись под ним.
        return { setState() {}, pulse() {}, destroy() {} };
    }

    const ctx = canvas.getContext('2d');
    const N = 380;              // частиц
    const NEIGHBOURS = 10;      // сколько соседей проверяем на линию
    const REDUCED = window.matchMedia &&
        window.matchMedia('(prefers-reduced-motion: reduce)').matches;

    // Состояния: цвет, радиус облака, скорость вращения, длина линии.
    const MODES = {
        idle:      { color: [132, 176, 202], radius: 1.00, spin: 0.12, link: 0.45, glow: 0.75 },
        listening: { color: [96, 186, 245],  radius: 1.14, spin: 0.24, link: 0.70, glow: 1.00 },
        thinking:  { color: [232, 172, 88],  radius: 0.92, spin: 0.78, link: 0.75, glow: 1.05 },
        speaking:  { color: [92, 208, 162],  radius: 0.86, spin: 0.36, link: 1.00, glow: 1.15 },
    };

    let mode = MODES.idle;
    // Текущие значения тянутся к целевым: переключение состояния должно
    // выглядеть как движение, а не как подмена картинки.
    const cur = { color: MODES.idle.color.slice(), radius: 1, link: 0.45, glow: 0.75 };

    let energy = 0;       // всплеск от pulse(), затухает сам
    let angle = 0;
    let raf = 0;
    let destroyed = false;
    let w = 0, h = 0, cx = 0, cy = 0, scale = 0;

    // ── Частицы на сфере ──
    // Равномерно по объёму: если брать радиус линейно, точки собьются
    // в центр и облако станет комком.
    const px = new Float32Array(N);
    const py = new Float32Array(N);
    const pz = new Float32Array(N);
    const phase = new Float32Array(N);

    for (let i = 0; i < N; i++) {
        const theta = Math.random() * Math.PI * 2;
        const phi = Math.acos(2 * Math.random() - 1);
        const r = Math.cbrt(Math.random()) * 0.6 + 0.4;
        px[i] = r * Math.sin(phi) * Math.cos(theta);
        py[i] = r * Math.sin(phi) * Math.sin(theta);
        pz[i] = r * Math.cos(phi);
        phase[i] = Math.random() * Math.PI * 2;
    }

    // Экранные координаты кадра — считаем один раз, используем и для
    // точек, и для линий.
    const sx = new Float32Array(N);
    const sy = new Float32Array(N);
    const sd = new Float32Array(N);   // «глубина» 0..1 для яркости

    function resize() {
        const rect = canvas.getBoundingClientRect();
        const dpr = Math.min(window.devicePixelRatio || 1, 2);
        w = Math.max(1, Math.round(rect.width));
        h = Math.max(1, Math.round(rect.height));
        canvas.width = Math.round(w * dpr);
        canvas.height = Math.round(h * dpr);
        ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
        cx = w / 2;
        cy = h / 2;
        scale = Math.min(w, h) * 0.40;
    }

    const ro = window.ResizeObserver ? new ResizeObserver(resize) : null;
    if (ro) ro.observe(canvas); else window.addEventListener('resize', resize);
    resize();

    function approach(from, to, k) { return from + (to - from) * k; }

    let last = performance.now();

    function frame(now) {
        if (destroyed) return;
        const dt = Math.min((now - last) / 1000, 0.05);
        last = now;

        // Плавный переход к параметрам состояния.
        for (let c = 0; c < 3; c++) {
            cur.color[c] = approach(cur.color[c], mode.color[c], 0.06);
        }
        cur.radius = approach(cur.radius, mode.radius, 0.05);
        cur.link = approach(cur.link, mode.link, 0.06);
        cur.glow = approach(cur.glow, mode.glow, 0.06);

        energy *= 0.90;
        angle += (mode.spin + energy * 1.6) * dt;

        const t = now / 1000;
        const cos = Math.cos(angle);
        const sin = Math.sin(angle);
        const breathe = REDUCED ? 0 : 0.05;
        const rad = scale * (cur.radius + energy * 0.10);

        const [cr, cg, cb] = cur.color;

        ctx.clearRect(0, 0, w, h);

        // ── Позиции ──
        for (let i = 0; i < N; i++) {
            // Лёгкое дыхание, у каждой частицы своя фаза: без него
            // облако выглядит жёстким телом, а не живым роем.
            const puff = 1 + Math.sin(t * 1.4 + phase[i]) * breathe;

            const x = px[i] * puff;
            const y = py[i] * puff;
            const z = pz[i] * puff;

            // Вращение вокруг вертикальной оси.
            const rx = x * cos - z * sin;
            const rz = x * sin + z * cos;

            // Слабая перспектива: дальние точки мельче и бледнее.
            const depth = 1 / (1.8 - rz * 0.55);
            sx[i] = cx + rx * rad * depth;
            sy[i] = cy + y * rad * depth;
            sd[i] = depth;
        }

        // ── Линии связи ──
        // Проверяем не все пары (это 70 тысяч сравнений на кадр), а
        // только ближайших по индексу соседей: частицы разложены по
        // сфере случайно, поэтому сетка получается такой же живой.
        const maxLen = scale * (0.30 + cur.link * 0.32);
        const maxLen2 = maxLen * maxLen;
        ctx.lineWidth = 1;
        for (let i = 0; i < N; i++) {
            for (let k = 1; k <= NEIGHBOURS; k++) {
                const j = (i + k) % N;
                const dx = sx[i] - sx[j];
                const dy = sy[i] - sy[j];
                const d2 = dx * dx + dy * dy;
                if (d2 > maxLen2) continue;
                const a = (1 - d2 / maxLen2) * 0.42 * cur.link * cur.glow;
                if (a < 0.012) continue;
                ctx.strokeStyle = `rgba(${cr | 0},${cg | 0},${cb | 0},${a.toFixed(3)})`;
                ctx.beginPath();
                ctx.moveTo(sx[i], sy[i]);
                ctx.lineTo(sx[j], sy[j]);
                ctx.stroke();
            }
        }

        // ── Частицы ──
        // lighter даёт то же свечение, что аддитивный блендинг в WebGL:
        // там, где точки накладываются, ядро облака светится ярче.
        ctx.globalCompositeOperation = 'lighter';
        for (let i = 0; i < N; i++) {
            const depth = sd[i];
            const a = Math.min(1, (depth - 0.40) * 2.6) * cur.glow;
            if (a <= 0.02) continue;
            const r = Math.max(0.8, depth * 2.9);
            ctx.fillStyle = `rgba(${cr | 0},${cg | 0},${cb | 0},${a.toFixed(3)})`;
            ctx.beginPath();
            ctx.arc(sx[i], sy[i], r, 0, Math.PI * 2);
            ctx.fill();
        }
        ctx.globalCompositeOperation = 'source-over';

        raf = requestAnimationFrame(frame);
    }
    raf = requestAnimationFrame(frame);

    return {
        setState(state) {
            mode = MODES[state] || MODES.idle;
        },
        // Толчок на слове: орб коротко раздувается и ускоряется.
        pulse() {
            energy = Math.min(1, energy + 0.45);
        },
        destroy() {
            destroyed = true;
            cancelAnimationFrame(raf);
            if (ro) ro.disconnect();
            else window.removeEventListener('resize', resize);
        },
    };
}
