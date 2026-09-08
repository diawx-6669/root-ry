# -*- coding: utf-8 -*-
"""
Генератор значков и знаков уровня КСПОЯ.

Значки хранятся в базе как идентификаторы (book, fire, rank_b1), а
рисуются этими файлами. Раньше значок БЫЛ эмодзи: одна и та же награда
выглядела по-разному в Windows, на телефоне и в проекторе актового зала.

Фон и ободок значка задаёт его редкость (см. rarity.py) — та же логика,
что у аватарок: в полосе значков профиля ценность должна читаться
раньше, чем сам предмет. У знаков уровня своя шкала: их фон идёт от
уровня, потому что уровень — это не редкость, а результат.
"""

import os

import rarity as R

OUT = os.path.join(os.path.dirname(__file__), "..", "static", "img", "badges")
OUT_RANK = os.path.join(os.path.dirname(__file__), "..", "static", "img", "ranks")


def disc(rarity, body):
    """Круглый значок: фон редкости, блик, фигура, ободок."""
    return (
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 96 96" '
        'width="96" height="96" role="img">'
        f"<defs>{R.gradient(rarity, 'bg')}"
        '<linearGradient id="shine" x1="0" y1="0" x2="0" y2="1">'
        '<stop offset="0%" stop-color="#ffffff" stop-opacity=".55"/>'
        '<stop offset="55%" stop-color="#ffffff" stop-opacity="0"/>'
        "</linearGradient>"
        '<clipPath id="card"><circle cx="48" cy="48" r="47"/></clipPath>'
        "</defs>"
        '<g clip-path="url(#card)">'
        '<circle cx="48" cy="48" r="47" fill="url(#bg)"/>'
        '<ellipse cx="48" cy="26" rx="42" ry="30" fill="url(#shine)"/>'
        f"{body}</g>"
        f'<circle cx="48" cy="48" r="45.6" fill="none" stroke="{R.ring(rarity)}" '
        'stroke-width="2.8" opacity=".7"/>'
        f'<circle cx="48" cy="48" r="41" fill="none" stroke="{R.ring(rarity)}" '
        'stroke-width="1.2" opacity=".28"/>'
        "</svg>"
    )


BADGES = {}

# ── Обычные ─────────────────────────────────────────────────────────
BADGES["book"] = disc(
    "common",
    '<path d="M16 26h20a10 10 0 0 1 12 6 10 10 0 0 1 12-6h20v42H60a10 10 0 0 0-12 6'
    ' 10 10 0 0 0-12-6H16z" fill="#5c6b7a"/>'
    '<path d="M48 32a10 10 0 0 1 12-6h20v42H60a10 10 0 0 0-12 6z" fill="#4a5766"/>'
    '<path d="M22 34h14M22 42h14M22 50h11M60 34h14M60 42h14M60 50h11"'
    ' stroke="#e6ecf2" stroke-width="3" stroke-linecap="round" opacity=".85"/>'
    '<rect x="45" y="26" width="6" height="48" rx="3" fill="#3c4856"/>'
    '<path d="M66 26h8v20l-4-5-4 5z" fill="#ef6b5c"/>',
)
BADGES["pencil"] = disc(
    "common",
    '<path d="M58 18l20 20-34 34-24 4 4-24z" fill="#f0b429"/>'
    '<path d="M58 18l20 20-17 17-20-20z" fill="#ffd166"/>'
    '<path d="M68 8a14 14 0 0 1 20 20l-10 10-20-20z" fill="#9aa7b4"/>'
    '<path d="M78 8a14 14 0 0 1 10 20l-10 10-10-10z" fill="#7c8996"/>'
    '<path d="M24 52l20 20-24 4z" fill="#fde9b8"/>'
    '<path d="M20 76l2-12 10 10z" fill="#38424e"/>'
    '<path d="M52 24l20 20" stroke="#e0a11a" stroke-width="2.4"'
    ' stroke-linecap="round" opacity=".7"/>',
)
BADGES["notepad"] = disc(
    "common",
    '<rect x="24" y="18" width="42" height="58" rx="7" fill="#5c6b7a"/>'
    '<rect x="24" y="18" width="12" height="58" fill="#4a5766"/>'
    '<path d="M30 14v10M38 14v10M46 14v10M54 14v10" stroke="#9aa7b4"'
    ' stroke-width="3.4" stroke-linecap="round"/>'
    '<path d="M42 34h18M42 44h18M42 54h12" stroke="#e6ecf2" stroke-width="3.4"'
    ' stroke-linecap="round" opacity=".85"/>'
    '<path d="M62 62l16-16 8 8-16 16-11 3z" fill="#f0b429"/>'
    '<path d="M78 46l8 8-4 4-8-8z" fill="#9aa7b4"/>'
    '<path d="M59 73l3-11 8 8z" fill="#fde9b8"/>',
)
BADGES["backpack"] = disc(
    "common",
    '<path d="M34 24a14 14 0 0 1 28 0" fill="none" stroke="#8b98a5"'
    ' stroke-width="5" stroke-linecap="round"/>'
    '<path d="M22 46a16 16 0 0 1 8-14v40h-8z" fill="#5c6b7a"/>'
    '<path d="M74 46a16 16 0 0 0-8-14v40h8z" fill="#4a5766"/>'
    '<path d="M28 44a20 20 0 0 1 40 0v32a6 6 0 0 1-6 6H34a6 6 0 0 1-6-6z"'
    ' fill="#68788a"/>'
    '<path d="M48 24a20 20 0 0 1 20 20v32a6 6 0 0 1-6 6H48z" fill="#566b7d"/>'
    '<path d="M28 44a20 20 0 0 1 40 0v10H28z" fill="#7b8c9e"/>'
    '<rect x="32" y="58" width="32" height="17" rx="5" fill="#e6ecf2"/>'
    '<path d="M32 66h32" stroke="#9aa7b4" stroke-width="2.6"/>'
    '<circle cx="56" cy="66" r="3.4" fill="#f0b429"/>'
    '<path d="M56 66v6" stroke="#f0b429" stroke-width="2.6"'
    ' stroke-linecap="round"/>'
    '<rect x="41" y="48" width="14" height="4" rx="2" fill="#e6ecf2"'
    ' opacity=".7"/>',
)

# ── Редкие ──────────────────────────────────────────────────────────
BADGES["star"] = disc(
    "rare",
    '<path d="M48 14l10 22 24 3-17 17 4 24-21-11-21 11 4-24-17-17 24-3z"'
    ' fill="#2f6fd0"/>'
    '<path d="M48 14l10 22 24 3-17 17 4 24-21-11z" fill="#2861bc"/>'
    '<path d="M48 24l7 15 16 2-12 12 3 16-14-8-14 8 3-16-12-12 16-2z"'
    ' fill="#5b9bf8"/>'
    '<path d="M48 24l7 15 16 2-12 12 3 16-14-8z" fill="#4a8bef"/>'
    '<circle cx="41" cy="38" r="3" fill="#fff" opacity=".7"/>',
)
BADGES["fire"] = disc(
    "rare",
    '<path d="M48 12c15 15 24 26 24 39a24 24 0 0 1-48 0c0-9 4-16 11-22 2 7 5 10 9 11'
    '-5-13 0-22 4-28z" fill="#e35a3a"/>'
    '<path d="M48 12c15 15 24 26 24 39a24 24 0 0 1-24 24z" fill="#cf4a2c"'
    ' opacity=".55"/>'
    '<path d="M48 40c8 9 12 15 12 21a12 12 0 0 1-24 0c0-6 4-12 12-21z"'
    ' fill="#ffb23f"/>'
    '<path d="M48 50c4 5 6 8 6 11a6 6 0 0 1-12 0c0-3 2-6 6-11z" fill="#ffe08a"/>',
)
BADGES["bulb"] = disc(
    "rare",
    '<path d="M48 14a22 22 0 0 1 13 39.6V60H35v-6.4A22 22 0 0 1 48 14z"'
    ' fill="#f6c453"/>'
    '<path d="M48 14a22 22 0 0 1 13 39.6V60H48z" fill="#e8ab2b"/>'
    '<path d="M42 26a12 12 0 0 1 10-4" stroke="#fff6d8" stroke-width="3.6"'
    ' fill="none" stroke-linecap="round" opacity=".8"/>'
    '<rect x="35" y="62" width="26" height="6" rx="3" fill="#8b98a5"/>'
    '<rect x="37" y="70" width="22" height="6" rx="3" fill="#6f7d8b"/>'
    '<path d="M48 34v18M42 42h12" stroke="#c8890f" stroke-width="2.6"'
    ' stroke-linecap="round" opacity=".55"/>'
    '<path d="M18 30l7 3M78 30l-7 3M24 12l5 6M72 12l-5 6" stroke="#5b9bf8"'
    ' stroke-width="3" stroke-linecap="round" opacity=".5"/>',
)

# ── Эпические ───────────────────────────────────────────────────────
BADGES["trophy"] = disc(
    "epic",
    '<path d="M30 16h36v18a18 18 0 0 1-36 0z" fill="#f0b429"/>'
    '<path d="M48 16h18v18a18 18 0 0 1-18 18z" fill="#d99a10"/>'
    '<path d="M30 20H20a12 12 0 0 0 12 14M66 20h10a12 12 0 0 1-12 14"'
    ' fill="none" stroke="#f0b429" stroke-width="4.6" stroke-linecap="round"/>'
    '<rect x="43" y="50" width="10" height="14" fill="#c98a08"/>'
    '<rect x="32" y="64" width="32" height="8" rx="3" fill="#8b3fd6"/>'
    '<rect x="27" y="72" width="42" height="9" rx="4" fill="#a855f7"/>'
    '<path d="M36 24q4 12 12 16" stroke="#ffe08a" stroke-width="3.4"'
    ' fill="none" stroke-linecap="round" opacity=".8"/>',
)
BADGES["diamond"] = disc(
    "epic",
    '<path d="M28 24h40l14 16-34 38-34-38z" fill="#9d4ff0"/>'
    '<path d="M48 24h20l14 16-34 38z" fill="#8a35e8"/>'
    '<path d="M28 24l6 16h28l6-16" fill="none" stroke="#e6d2ff"'
    ' stroke-width="2.6" opacity=".85"/>'
    '<path d="M14 40h68" stroke="#e6d2ff" stroke-width="2.6" opacity=".85"/>'
    '<path d="M48 78L34 40M48 78l14-38" stroke="#e6d2ff" stroke-width="2.6"'
    ' opacity=".7"/>'
    '<path d="M34 28h12l-4 10h-6z" fill="#fff" opacity=".4"/>',
)

# ── Легендарные ─────────────────────────────────────────────────────
BADGES["crown"] = disc(
    "legendary",
    '<path d="M18 66L12 26l22 15 14-24 14 24 22-15-6 40z" fill="#f0a91f"/>'
    '<path d="M48 17l14 24 22-15-6 40H48z" fill="#d98c0c"/>'
    '<rect x="18" y="66" width="60" height="11" rx="4.5" fill="#c47b06"/>'
    '<rect x="18" y="66" width="60" height="4" rx="2" fill="#ffcd63"'
    ' opacity=".65"/>'
    '<circle cx="31" cy="50" r="4.2" fill="#fff3c4"/>'
    '<circle cx="48" cy="44" r="5.2" fill="#fff3c4"/>'
    '<circle cx="65" cy="50" r="4.2" fill="#fff3c4"/>'
    '<circle cx="12" cy="26" r="4" fill="#ffe08a"/>'
    '<circle cx="48" cy="17" r="4.6" fill="#ffe08a"/>'
    '<circle cx="84" cy="26" r="4" fill="#ffe08a"/>',
)

# ── Особый: дерево грамматики целиком ───────────────────────────────
# Из кейсов не выпадает, поэтому берёт мифический тон: единственная
# награда, которую нельзя купить.
BADGES["tree"] = disc(
    "mythic",
    '<circle cx="48" cy="32" r="17" fill="#2f9e5f"/>'
    '<circle cx="31" cy="45" r="13" fill="#37b06c"/>'
    '<circle cx="65" cy="45" r="13" fill="#37b06c"/>'
    '<circle cx="48" cy="47" r="15" fill="#43c079"/>'
    '<circle cx="41" cy="28" r="6" fill="#5ed092" opacity=".55"/>'
    '<rect x="43" y="52" width="10" height="26" rx="4" fill="#8a5a2b"/>'
    '<path d="M48 62L36 54M48 68l12-8" stroke="#8a5a2b" stroke-width="4.4"'
    ' stroke-linecap="round"/>'
    '<circle cx="34" cy="38" r="3" fill="#ffe08a"/>'
    '<circle cx="62" cy="34" r="2.6" fill="#ffe08a"/>',
)

# ── Уровни КСПОЯ ────────────────────────────────────────────────────
#
# Шкала своя: уровень — не редкость, а результат теста. Тон растёт от
# серого A1 к малиновому C2, а число засечек по краю показывает ступень:
# на маленькой иконке в списке цвет различить труднее, чем количество.
RANKS = {
    "rank_a1": ("#f2f4f6", "#dfe4e9", "#8794a1", "#5b6875", "A1", 1),
    "rank_a2": ("#e9f7ee", "#c7ead4", "#43b06c", "#1f7a45", "A2", 2),
    "rank_b1": ("#e9f2ff", "#c6dcfb", "#5b9bf8", "#1f5fbe", "B1", 3),
    "rank_b2": ("#fff6e3", "#ffe3a6", "#f0a91f", "#a9700a", "B2", 4),
    "rank_c1": ("#f5edff", "#dfcbfb", "#b072f5", "#7a34c7", "C1", 5),
    "rank_c2": ("#ffe9f1", "#ffc9dd", "#f4649a", "#b3245a", "C2", 6),
}


def rank(inner, outer, ring, ink, label, step):
    # Засечки по верхней дуге: закрашено столько, сколько ступеней взято.
    pips = []
    for i in range(6):
        angle = -140 + i * 20
        import math
        a = math.radians(angle)
        x, y = 48 + 37 * math.cos(a), 48 + 37 * math.sin(a)
        on = i < step
        pips.append(
            f'<circle cx="{x:.1f}" cy="{y:.1f}" r="{3.2 if on else 2.4}" '
            f'fill="{ring if on else ink}" opacity="{1 if on else .18}"/>'
        )
    return (
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 96 96" '
        'width="96" height="96" role="img">'
        "<defs>"
        f'<radialGradient id="bg" cx="50%" cy="38%" r="78%">'
        f'<stop offset="0%" stop-color="{inner}"/>'
        f'<stop offset="100%" stop-color="{outer}"/>'
        "</radialGradient>"
        '<linearGradient id="shine" x1="0" y1="0" x2="0" y2="1">'
        '<stop offset="0%" stop-color="#ffffff" stop-opacity=".5"/>'
        '<stop offset="55%" stop-color="#ffffff" stop-opacity="0"/>'
        "</linearGradient>"
        '<clipPath id="card"><circle cx="48" cy="48" r="47"/></clipPath>'
        "</defs>"
        '<g clip-path="url(#card)">'
        '<circle cx="48" cy="48" r="47" fill="url(#bg)"/>'
        '<ellipse cx="48" cy="26" rx="42" ry="30" fill="url(#shine)"/>'
        f'<circle cx="48" cy="50" r="27" fill="{ring}" opacity=".14"/>'
        f'<path d="M48 80l-9 11-1-16z" fill="{ring}" opacity=".55"/>'
        f'<path d="M48 80l9 11 1-16z" fill="{ring}" opacity=".38"/>'
        + "".join(pips)
        + f'<text x="48" y="60" text-anchor="middle" font-family="Montserrat, '
        f'Segoe UI, Arial, sans-serif" font-size="27" font-weight="700" '
        f'fill="{ink}">{label}</text>'
        "</g>"
        f'<circle cx="48" cy="48" r="45.6" fill="none" stroke="{ring}" '
        'stroke-width="2.8" opacity=".75"/>'
        "</svg>"
    )


def main():
    os.makedirs(OUT, exist_ok=True)
    os.makedirs(OUT_RANK, exist_ok=True)
    for name, svg in BADGES.items():
        open(os.path.join(OUT, name + ".svg"), "w", encoding="utf-8").write(svg)
    for name, args in RANKS.items():
        open(os.path.join(OUT_RANK, name + ".svg"), "w", encoding="utf-8").write(
            rank(*args)
        )
    print("значков:", len(BADGES), "уровней:", len(RANKS))


if __name__ == "__main__":
    main()
