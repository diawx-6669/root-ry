# -*- coding: utf-8 -*-
"""
Генератор аватарок RootRy.

Раньше аватарки грузились с api.dicebear.com, а запасным вариантом был
эмодзи, нарисованный в <text>: без интернета вместо картинки появлялся
системный шрифт, а на защите — вообще квадрат. Здесь аватарки собираются
один раз в статические SVG, лежат рядом с сайтом и не зависят ни от сети,
ни от эмодзи-шрифта.

Фон и ободок задаёт редкость, а не зверь (см. rarity.py): в магазине и в
профиле ценность предмета должна читаться раньше, чем сам предмет.
Поэтому под фигурой лежит светлое пятно — иначе синий дельфин на синем
фоне редкой рамки сливался бы с ней.
"""

import math
import os

import rarity as R

OUT = os.path.join(os.path.dirname(__file__), "..", "static", "img", "avatars")

# Голова: мягкий блок с широкими щеками и чуть приплюснутым лбом.
HEAD = "M64 28c21 0 35 14.5 35 34 0 21.5-15.5 36-35 36S29 83.5 29 62c0-19.5 14-34 35-34z"


# ── Каркас картинки ─────────────────────────────────────────────────

def frame(rarity, body):
    """Собрать аватарку: фон редкости, пятно света, тень, фигура, ободок.

    Карточка круглая, а не квадратная со скруглением: в шапке, рейтинге,
    кабинете учителя и пикере аватарка обрезается по кругу (border-radius:
    50%). У квадратной картинки при такой обрезке срезался бы ободок
    редкости — то самое, ради чего он и нужен.
    """
    return (
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 128 128" '
        'width="128" height="128" role="img">'
        f"<defs>{R.gradient(rarity, 'bg')}"
        '<radialGradient id="halo" cx="50%" cy="50%" r="50%">'
        '<stop offset="55%" stop-color="#ffffff" stop-opacity=".42"/>'
        '<stop offset="100%" stop-color="#ffffff" stop-opacity="0"/>'
        "</radialGradient>"
        '<clipPath id="card"><circle cx="64" cy="64" r="64"/></clipPath>'
        "</defs>"
        '<g clip-path="url(#card)">'
        '<circle cx="64" cy="64" r="64" fill="url(#bg)"/>'
        '<circle cx="64" cy="62" r="52" fill="url(#halo)"/>'
        '<ellipse cx="64" cy="105" rx="31" ry="6" fill="#0b2233" opacity=".09"/>'
        # Фигура ужимается к центру: круглая обрезка иначе срезала бы то,
        # что торчит вбок и вверх — рога дракона, уши кота, поля шляпы.
        f'<g transform="translate(64 64) scale(.9) translate(-64 -64)">{body}</g>'
        "</g>"
        f'<circle cx="64" cy="64" r="62.4" fill="none" '
        f'stroke="{R.ring(rarity)}" stroke-width="3.2" opacity=".7"/>'
        "</svg>"
    )


# ── Детали фигуры ───────────────────────────────────────────────────

def head(fill, shade=None):
    """Голова с боковой тенью и верхним бликом — даёт объём без обводки."""
    out = f'<path d="{HEAD}" fill="{fill}"/>'
    if shade:
        out += (
            f'<path d="M64 28c21 0 35 14.5 35 34 0 21.5-15.5 36-35 36z" '
            f'fill="{shade}" opacity=".38"/>'
        )
    out += '<ellipse cx="51" cy="46" rx="15" ry="10" fill="#fff" opacity=".2"/>'
    return out


def eyes(y=64, dx=14.5, r=5.2, color="#22303a"):
    return (
        f'<ellipse cx="{64-dx}" cy="{y}" rx="{r}" ry="{r*1.08:.2f}" fill="{color}"/>'
        f'<ellipse cx="{64+dx}" cy="{y}" rx="{r}" ry="{r*1.08:.2f}" fill="{color}"/>'
        f'<circle cx="{64-dx+1.9}" cy="{y-2.1}" r="1.8" fill="#fff" opacity=".95"/>'
        f'<circle cx="{64+dx+1.9}" cy="{y-2.1}" r="1.8" fill="#fff" opacity=".95"/>'
        f'<circle cx="{64-dx-1.7}" cy="{y+2.2}" r="1" fill="#fff" opacity=".5"/>'
        f'<circle cx="{64+dx-1.7}" cy="{y+2.2}" r="1" fill="#fff" opacity=".5"/>'
    )


def blush(y=78, dx=25, color="#f2867f", opacity=".3"):
    return (
        f'<ellipse cx="{64-dx}" cy="{y}" rx="7" ry="4.4" fill="{color}" opacity="{opacity}"/>'
        f'<ellipse cx="{64+dx}" cy="{y}" rx="7" ry="4.4" fill="{color}" opacity="{opacity}"/>'
    )


def muzzle(fill, nose="#4d3529", y=79, w=20, h=13.5, smile=True):
    out = (
        f'<ellipse cx="64" cy="{y}" rx="{w}" ry="{h}" fill="{fill}"/>'
        f'<ellipse cx="64" cy="{y-1}" rx="{w-3}" ry="{h-3}" fill="#fff" opacity=".22"/>'
        f'<path d="M57.5 {y-4.5}h13a2.6 2.6 0 0 1 2 4.3l-6.5 5.4a2.6 2.6 0 0 1-4 0'
        f'l-6.5-5.4a2.6 2.6 0 0 1 2-4.3z" fill="{nose}"/>'
    )
    if smile:
        out += (
            f'<path d="M64 {y+5}v3.4M64 {y+8.4}q-4.4 3.4-8-.6M64 {y+8.4}q4.4 3.4 8-.6" '
            f'stroke="{nose}" stroke-width="2.4" fill="none" stroke-linecap="round"/>'
        )
    return out


def round_ears(fill, inner, cx=27, cy=40, r=13.5):
    return (
        f'<circle cx="{64-cx}" cy="{cy}" r="{r}" fill="{fill}"/>'
        f'<circle cx="{64+cx}" cy="{cy}" r="{r}" fill="{fill}"/>'
        f'<circle cx="{64-cx}" cy="{cy+1}" r="{r*0.5:.1f}" fill="{inner}"/>'
        f'<circle cx="{64+cx}" cy="{cy+1}" r="{r*0.5:.1f}" fill="{inner}"/>'
    )


def pointy_ears(fill, inner, spread=25, top=16):
    return (
        f'<path d="M{64-spread-15} {top+29}L{64-spread} {top}l15 24z" fill="{fill}"/>'
        f'<path d="M{64+spread+15} {top+29}L{64+spread} {top}l-15 24z" fill="{fill}"/>'
        f'<path d="M{64-spread-7.5} {top+25}L{64-spread+1} {top+10}l7.5 14z" fill="{inner}"/>'
        f'<path d="M{64+spread+7.5} {top+25}L{64+spread-1} {top+10}l-7.5 14z" fill="{inner}"/>'
    )


def tall_ears(fill, inner):
    return (
        f'<ellipse cx="44" cy="28" rx="10.5" ry="19" fill="{fill}" transform="rotate(-9 44 28)"/>'
        f'<ellipse cx="84" cy="28" rx="10.5" ry="19" fill="{fill}" transform="rotate(9 84 28)"/>'
        f'<ellipse cx="44" cy="30" rx="5" ry="11.5" fill="{inner}" transform="rotate(-9 44 30)"/>'
        f'<ellipse cx="84" cy="30" rx="5" ry="11.5" fill="{inner}" transform="rotate(9 84 30)"/>'
    )


def creature(rarity, fur, shade, ear, ear_fill, ear_inner, muzzle_fill,
             before="", after="", blush_on=True, nose="#4d3529"):
    parts = [before]
    if ear == "round":
        parts.append(round_ears(ear_fill, ear_inner))
    elif ear == "pointy":
        parts.append(pointy_ears(ear_fill, ear_inner))
    elif ear == "tall":
        parts.append(tall_ears(ear_fill, ear_inner))
    parts.append(head(fur, shade))
    parts.append(after)
    if blush_on:
        parts.append(blush())
    parts.append(eyes())
    if muzzle_fill:
        parts.append(muzzle(muzzle_fill, nose))
    return frame(rarity, "".join(parts))


AVATARS = {}

# ── Обычные: серо-стальная рамка ────────────────────────────────────
AVATARS["cat"] = creature(
    "common", "#98a6b5", "#7d8b9a", "pointy", "#98a6b5", "#f0b0bd", "#eef2f6",
    after='<path d="M20 71h20M20 80h19M108 71H88M108 80H89" stroke="#5b6875"'
          ' stroke-width="2.6" stroke-linecap="round" opacity=".75"/>',
)
AVATARS["dog"] = creature(
    "common", "#c98b5e", "#a86f47", "tall", "#a06437", "#dda57a", "#f3ddc9",
)
AVATARS["fox"] = creature(
    "common", "#e8763c", "#c85c28", "pointy", "#e8763c", "#ffd9c2", "#fdf3ea",
    after='<path d="M29 61c8-4 18-5 35-5s27 1 35 5c-2 16-16 28-35 28S31 77 29 61z"'
          ' fill="#f4a06f" opacity=".28"/>',
)
AVATARS["panda"] = creature(
    "common", "#fbfbfc", "#dfe3e8", "round", "#2f3336", "#565c61", "#ffffff",
    after='<ellipse cx="49" cy="64" rx="12.5" ry="14" fill="#2f3336"'
          ' transform="rotate(-12 49 64)"/>'
          '<ellipse cx="79" cy="64" rx="12.5" ry="14" fill="#2f3336"'
          ' transform="rotate(12 79 64)"/>',
    blush_on=False,
)
AVATARS["koala"] = creature(
    "common", "#a4b0b9", "#8994a0", None, None, None, "#e7ecf0",
    before='<circle cx="22" cy="52" r="18" fill="#a4b0b9"/>'
           '<circle cx="106" cy="52" r="18" fill="#a4b0b9"/>'
           '<circle cx="22" cy="53" r="10" fill="#cfd8de"/>'
           '<circle cx="106" cy="53" r="10" fill="#cfd8de"/>',
    nose="#3c4a55",
)
AVATARS["bear"] = creature(
    "common", "#a9713f", "#8b5a30", "round", "#8b5a30", "#c99062", "#e8cdb2",
)
AVATARS["frog"] = creature(
    "common", "#57b85a", "#41984a", None, None, None, None,
    before='<circle cx="41" cy="35" r="15" fill="#57b85a"/>'
           '<circle cx="87" cy="35" r="15" fill="#57b85a"/>',
    after='<circle cx="41" cy="34" r="8" fill="#fff"/>'
          '<circle cx="87" cy="34" r="8" fill="#fff"/>'
          '<circle cx="41" cy="35" r="4" fill="#22303a"/>'
          '<circle cx="87" cy="35" r="4" fill="#22303a"/>'
          '<circle cx="42.6" cy="33" r="1.7" fill="#fff"/>'
          '<circle cx="88.6" cy="33" r="1.7" fill="#fff"/>'
          '<path d="M44 74q20 15 40 0" stroke="#2c7a35" stroke-width="3.6"'
          ' fill="none" stroke-linecap="round"/>'
          '<circle cx="52" cy="66" r="2.4" fill="#3f9b46"/>'
          '<circle cx="76" cy="68" r="2" fill="#3f9b46"/>',
)
AVATARS["tiger"] = creature(
    "common", "#f0973f", "#d0761f", "round", "#e58b33", "#ffd6b3", "#fbe8d5",
    after='<path d="M45 39l5 17M64 33v21M83 39l-5 17M31 60l11 5M97 60l-11 5"'
          ' stroke="#5c3110" stroke-width="4.6" stroke-linecap="round"/>',
)

# Лев: грива кольцом вокруг головы.
_mane = "".join(
    f'<circle cx="{64 + 40 * math.cos(a):.1f}" cy="{66 + 39 * math.sin(a):.1f}" '
    f'r="14" fill="#c97b28"/>'
    for a in [i * math.pi / 6 for i in range(12)]
) + "".join(
    f'<circle cx="{64 + 40 * math.cos(a):.1f}" cy="{66 + 39 * math.sin(a):.1f}" '
    f'r="9" fill="#e09a45" opacity=".7"/>'
    for a in [(i + 0.5) * math.pi / 6 for i in range(12)]
)
AVATARS["lion"] = creature(
    "common", "#f0a94a", "#d78c2c", None, None, None, "#f8dfb4",
    before=_mane,
)

# ── Редкие: синяя рамка ─────────────────────────────────────────────
AVATARS["unicorn"] = creature(
    "rare", "#fbf7ff", "#e4d9f2", "pointy", "#fbf7ff", "#e6d3f7", "#f4ecfd",
    before='<path d="M64 2l10 30H54z" fill="#ffd166"/>'
           '<path d="M60 26l7-22 3 7-6 17z" fill="#e8a92c"/>'
           '<path d="M84 26q14-6 20 6-12 2-17 10z" fill="#c4a8ef"/>',
    nose="#8a6fa8",
)
AVATARS["dragon"] = creature(
    "rare", "#3fa66f", "#2d8055", None, None, None, "#c6e8d4",
    # Рога и гребень рисуются ДО головы, но выше её верхнего края:
    # раньше они начинались на той же высоте и прятались за черепом.
    before='<path d="M30 8l14 22-20 4z" fill="#1f6b46"/>'
           '<path d="M98 8L84 30l20 4z" fill="#1f6b46"/>'
           '<path d="M33 12l9 16-13 2z" fill="#2d8055"/>'
           '<path d="M95 12l-9 16 13 2z" fill="#2d8055"/>'
           '<path d="M46 30l6-14 6 12 6-14 6 14 6-12 6 14z" fill="#1f6b46"/>',
    after='<path d="M46 30l6-14 6 12 6-14 6 14 6-12 6 14z" fill="#2d8055"'
          ' opacity=".55"/>'
          '<ellipse cx="46" cy="56" rx="7" ry="5" fill="#57bd85" opacity=".45"/>'
          '<ellipse cx="82" cy="56" rx="7" ry="5" fill="#57bd85" opacity=".45"/>',
    nose="#1f5c3c",
)
AVATARS["butterfly"] = frame(
    "rare",
    '<path d="M60 46c-10-19-34-25-43-9-9 17 8 34 30 36-16 10-23 26-11 35 12 9 23-8 25-26z"'
    ' fill="#7c4ff0"/>'
    '<path d="M60 46c-10-19-34-25-43-9 14-4 30 2 43 9z" fill="#fff" opacity=".22"/>'
    '<path d="M68 46c10-19 34-25 43-9 9 17-8 34-30 36 16 10 23 26 11 35-12 9-23-8-25-26z"'
    ' fill="#9d74f7"/>'
    '<path d="M68 46c10-19 34-25 43-9-14-4-30 2-43 9z" fill="#fff" opacity=".22"/>'
    '<circle cx="34" cy="42" r="6" fill="#fff" opacity=".55"/>'
    '<circle cx="94" cy="42" r="6" fill="#fff" opacity=".55"/>'
    '<rect x="60" y="38" width="8" height="60" rx="4" fill="#3f327a"/>'
    '<circle cx="64" cy="35" r="8" fill="#3f327a"/>'
    '<path d="M59 29l-9-12M69 29l9-12" stroke="#3f327a" stroke-width="3.2"'
    ' stroke-linecap="round"/>'
    '<circle cx="50" cy="17" r="2.6" fill="#3f327a"/>'
    '<circle cx="78" cy="17" r="2.6" fill="#3f327a"/>',
)
AVATARS["peacock"] = frame(
    "rare",
    # Веер: перья расходятся из основания хвоста, у каждого — «глазок».
    "".join(
        f'<ellipse cx="{64 + 40 * math.cos(a):.1f}" cy="{92 + 40 * math.sin(a):.1f}" '
        f'rx="9" ry="17" fill="#0e7490" '
        f'transform="rotate({math.degrees(a) + 90:.1f} '
        f'{64 + 40 * math.cos(a):.1f} {92 + 40 * math.sin(a):.1f})"/>'
        for a in [math.radians(200 + i * 28) for i in range(6)]
    )
    + "".join(
        f'<circle cx="{64 + 44 * math.cos(a):.1f}" cy="{92 + 44 * math.sin(a):.1f}" '
        f'r="5.5" fill="#22d3ee"/>'
        f'<circle cx="{64 + 44 * math.cos(a):.1f}" cy="{92 + 44 * math.sin(a):.1f}" '
        f'r="2.6" fill="#164e63"/>'
        for a in [math.radians(200 + i * 28) for i in range(6)]
    )
    + '<ellipse cx="64" cy="93" rx="17" ry="21" fill="#1197b4"/>'
    '<ellipse cx="58" cy="88" rx="8" ry="12" fill="#3fd0e8" opacity=".45"/>'
    '<rect x="58" y="52" width="12" height="34" rx="6" fill="#1197b4"/>'
    '<circle cx="64" cy="48" r="13" fill="#12a2c0"/>'
    '<circle cx="59" cy="43" r="4.5" fill="#5fdcf0" opacity=".5"/>'
    '<path d="M64 24l3 6 6 1-5 4 1 6-5-3-5 3 1-6-5-4 6-1z" fill="#3fd0e8"/>'
    '<path d="M64 30v6" stroke="#3fd0e8" stroke-width="2.4" stroke-linecap="round"/>'
    '<circle cx="69" cy="46" r="4" fill="#fff"/>'
    '<circle cx="69" cy="46" r="2" fill="#0b3a4a"/>'
    '<path d="M76 50l11 3-11 4z" fill="#ffd166"/>',
)
AVATARS["parrot"] = frame(
    "rare",
    # Профиль вправо: круглая голова, красная шапочка, жёлтый хохолок и
    # крючковатый клюв. Раньше клюв был отдельным пятном сбоку и голова
    # читалась просто зелёным кругом.
    '<path d="M92 84q16 12 16 26-14-2-22-14z" fill="#149146"/>'
    '<ellipse cx="56" cy="78" rx="30" ry="26" fill="#1faa53"/>'
    '<circle cx="58" cy="54" r="31" fill="#23c55f"/>'
    '<path d="M29 50a31 31 0 0 1 56-8l-8 12a22 22 0 0 0-40 6z" fill="#ef4444"/>'
    '<path d="M60 21l14-11-2 15z" fill="#ffd166"/>'
    '<path d="M45 21L33 8l16 5z" fill="#ffd166"/>'
    '<path d="M85 46q17 3 17 15 0 13-15 15-5-6-5-15z" fill="#fb923c"/>'
    '<path d="M85 62q10 3 12 10-6 4-12 4-2-7 0-14z" fill="#c2410c"/>'
    '<ellipse cx="44" cy="86" rx="16" ry="12" fill="#1a9c4a"'
    ' transform="rotate(-14 44 86)"/>'
    '<ellipse cx="42" cy="84" rx="9" ry="6" fill="#3ad477" opacity=".55"'
    ' transform="rotate(-14 42 84)"/>'
    '<circle cx="71" cy="47" r="9" fill="#fff"/>'
    '<circle cx="72" cy="47" r="4.4" fill="#22303a"/>'
    '<circle cx="74" cy="45" r="1.7" fill="#fff"/>'
    '<circle cx="40" cy="66" r="6" fill="#f2867f" opacity=".26"/>',
)
AVATARS["flamingo"] = frame(
    "rare",
    '<path d="M60 108c-3-26 1-40 12-51 11-11 13-24 4-32" fill="none"'
    ' stroke="#f472b6" stroke-width="14" stroke-linecap="round"/>'
    '<ellipse cx="54" cy="90" rx="28" ry="19" fill="#f9a8d4"/>'
    '<path d="M30 84q22-10 44 2-20 10-44-2z" fill="#fbcfe4"/>'
    '<circle cx="68" cy="27" r="17" fill="#f472b6"/>'
    '<circle cx="63" cy="21" r="6" fill="#fff" opacity=".28"/>'
    '<path d="M53 24l-24 8 24 11z" fill="#28323d"/>'
    '<path d="M53 30l-16 3 16 5z" fill="#4a5764"/>'
    '<circle cx="72" cy="23" r="4.6" fill="#fff"/>'
    '<circle cx="72" cy="23" r="2.4" fill="#22303a"/>',
)
AVATARS["dolphin"] = frame(
    "rare",
    '<path d="M14 100c6-36 30-60 66-63 20-2 34 5 40 13-11 1-19 7-23 15 10 9 13 22 8 34'
    '-12-5-21-16-25-26-24 10-46 18-66 27z" fill="#1e90d2"/>'
    '<path d="M60 44c-18 9-31 24-38 44 24-5 43-16 55-29-4-6-10-11-17-15z"'
    ' fill="#5cc0f0"/>'
    '<path d="M66 36l-8-22 24 16z" fill="#1878b0"/>'
    '<circle cx="92" cy="52" r="5" fill="#0a3550"/>'
    '<circle cx="94" cy="50" r="1.8" fill="#fff"/>'
    '<path d="M106 45c7-1 12 1 14 5-6 2-11 2-14-1z" fill="#0d6ea5"/>'
    '<path d="M78 66q12 4 20 0" stroke="#0a3550" stroke-width="2.6" fill="none"'
    ' stroke-linecap="round" opacity=".45"/>',
)

# ── Эпические: фиолетовая рамка ─────────────────────────────────────
AVATARS["wizard"] = frame(
    "epic",
    head("#f3d5b5", "#dcb894")
    + '<path d="M26 52q10-30 24-40 6 22 22 40z" fill="#6d28d9"/>'
    '<path d="M50 12q8 20 22 40H64z" fill="#5b21b6"/>'
    '<path d="M26 52q18-9 40-6 20 2 34 8-2 10-8 12-30-8-58-4-8-4-8-10z"'
    ' fill="#4c1d95"/>'
    '<path d="M26 52q18-9 40-6 20 2 34 8l-2 5q-16-6-34-8-22-2-38 6z"'
    ' fill="#7c3aed" opacity=".55"/>'
    '<path d="M50 8l3.4 9 9 3.4-9 3.4L50 33l-3.4-9.2-9-3.4 9-3.4z"'
    ' fill="#fbbf24"/>'
    + eyes(y=68, dx=13.5, r=4.6)
    + blush(y=79, dx=25, color="#dc9080", opacity=".26")
    + '<path d="M36 76q28 20 56 0 2 20-8 32-8 10-20 10t-20-10q-10-12-8-32z"'
    ' fill="#eef0f2"/>'
    '<path d="M36 76q28 20 56 0-4 8-28 10T36 76z" fill="#d7dade"/>'
    '<path d="M52 80q12 8 24 0 0 6-12 6t-12-6z" fill="#c9ccd2"/>'
    '<path d="M58 92q6 5 12 0" stroke="#b9bdc4" stroke-width="2.4" fill="none"'
    ' stroke-linecap="round"/>',
)
AVATARS["vampire"] = frame(
    "epic",
    head("#e9dde6", "#cfc0cc")
    + '<path d="M27 50q37-30 74 0l-9-16q-28-16-56 0z" fill="#1f2937"/>'
    '<path d="M51 33l13 13 13-13-7-7H58z" fill="#1f2937"/>'
    + eyes(y=66, dx=14, r=4.8, color="#8f1d2c")
    + blush(y=79, dx=25, color="#c04a5c", opacity=".22")
    + '<path d="M50 82h28q-3 13-14 13T50 82z" fill="#9f1239"/>'
    '<path d="M53 82h6.5l-3.2 9zM68.5 82H75l-3.2 9z" fill="#fff"/>'
    '<path d="M53 82h22" stroke="#fff" stroke-width="2.4" stroke-linecap="round"/>',
)
AVATARS["hero"] = frame(
    "epic",
    head("#f3d5b5", "#dcb894")
    + '<path d="M27 46q37-28 74 0l-6-13q-31-20-62 0z" fill="#7c2d12"/>'
    '<path d="M27 56h32l-3 12-26-4zM69 56h32l-3 9-26 3z" fill="#1d4ed8"/>'
    '<path d="M59 57h10v5H59z" fill="#1d4ed8"/>'
    '<path d="M27 56h32l-1 4-31-1z" fill="#3b82f6"/>'
    '<path d="M101 56H69l1 4 31-1z" fill="#3b82f6"/>'
    + eyes(y=64, dx=14, r=4.4)
    + '<path d="M50 84q14 12 28 0" stroke="#bd7550" stroke-width="3.4"'
    ' fill="none" stroke-linecap="round"/>',
)
AVATARS["elf"] = frame(
    "epic",
    # Голова занимает x от 29 до 99, поэтому уши уходят за её край:
    # нарисованные ближе, они целиком прятались под щеками.
    '<path d="M34 72L8 30l26 14z" fill="#f0d9c0"/>'
    '<path d="M94 72l26-42-26 14z" fill="#f0d9c0"/>'
    '<path d="M34 72L14 40l18 10z" fill="#dbbfa2"/>'
    '<path d="M94 72l20-32-18 10z" fill="#dbbfa2"/>'
    + head("#f0d9c0", "#d9bda0")
    + '<path d="M27 50q37-30 74 0l-7-16q-30-19-60 0z" fill="#d9a441"/>'
    '<path d="M27 50q10-14 22-19-4 12-4 22z" fill="#eab959"/>'
    '<path d="M92 34q10 8 9 20l-8-4z" fill="#eab959"/>'
    + eyes(y=66, dx=14, r=4.4)
    + blush(y=79, color="#dc9080", opacity=".26")
    + '<path d="M52 84q12 10 24 0" stroke="#c08a63" stroke-width="3.2"'
    ' fill="none" stroke-linecap="round"/>',
)
AVATARS["mermaid"] = frame(
    "epic",
    '<path d="M22 60q42-36 84 0v34q-6-16-16-22 4 14 2 26-9-24-28-24t-28 24'
    'q-2-12 2-26-10 6-16 22z" fill="#0d7f96"/>'
    + head("#f7d9c4", "#e0bda6")
    + '<path d="M27 48q37-30 74 0l-8-15q-29-18-58 0z" fill="#0d7f96"/>'
    '<path d="M27 48q11-13 24-18-6 10-7 20z" fill="#25aac2"/>'
    '<path d="M86 32q12 10 13 22l-9-5z" fill="#25aac2"/>'
    + eyes(y=66, dx=14, r=4.4)
    + blush(y=79, color="#e08a76", opacity=".28")
    + '<path d="M52 84q12 10 24 0" stroke="#c98f70" stroke-width="3.2"'
    ' fill="none" stroke-linecap="round"/>'
    '<path d="M84 112l14-20 6 12 8-8-2 22z" fill="#22d3ee"/>'
    '<path d="M92 101l6-9 4 8z" fill="#8ceaf7"/>',
)

# ── Легендарные: янтарная рамка ─────────────────────────────────────
AVATARS["crown"] = frame(
    "legendary",
    '<path d="M20 86L13 34l27 19 24-32 24 32 27-19-7 52z" fill="#f0a91f"/>'
    '<path d="M64 21l24 32 27-19-7 52H64z" fill="#d98c0c"/>'
    '<rect x="20" y="86" width="88" height="15" rx="6" fill="#c47b06"/>'
    '<rect x="20" y="86" width="88" height="6" rx="3" fill="#ffcd63" opacity=".6"/>'
    '<circle cx="40" cy="66" r="5.5" fill="#fff3c4"/>'
    '<circle cx="64" cy="59" r="7" fill="#fff3c4"/>'
    '<circle cx="88" cy="66" r="5.5" fill="#fff3c4"/>'
    '<circle cx="13" cy="34" r="5" fill="#ffe08a"/>'
    '<circle cx="64" cy="21" r="6" fill="#ffe08a"/>'
    '<circle cx="115" cy="34" r="5" fill="#ffe08a"/>',
)
AVATARS["star"] = frame(
    "legendary",
    '<path d="M64 12l15 33 36 4-27 25 8 36-32-19-32 19 8-36-27-25 36-4z"'
    ' fill="#f0a91f"/>'
    '<path d="M64 12l15 33 36 4-27 25 8 36-32-19z" fill="#d98c0c"/>'
    '<path d="M64 28l10 22 24 3-18 17 5 24-21-13-21 13 5-24-18-17 24-3z"'
    ' fill="#ffd977"/>'
    '<path d="M64 28l10 22 24 3-18 17 5 24-21-13z" fill="#ffc957"/>',
)
AVATARS["comet"] = frame(
    "legendary",
    # Хвост расширяется ОТ головы, как у настоящей кометы. Узкая полоса
    # одинаковой ширины читалась как палочка от леденца.
    '<path d="M76 22Q34 50 0 106Q12 122 34 130Q66 80 86 44Z" fill="#f0a91f"'
    ' opacity=".24"/>'
    '<path d="M78 28Q46 52 16 100Q26 114 44 122Q68 82 84 50Z" fill="#f0a91f"'
    ' opacity=".42"/>'
    '<path d="M80 34Q58 54 36 94Q44 104 56 110Q72 82 84 56Z" fill="#ffd977"'
    ' opacity=".72"/>'
    '<path d="M74 36Q56 54 40 88" stroke="#fff" stroke-width="3.4" fill="none"'
    ' stroke-linecap="round" opacity=".45"/>'
    '<circle cx="92" cy="36" r="24" fill="#f0a91f"/>'
    '<circle cx="92" cy="36" r="24" fill="none" stroke="#ffe08a"'
    ' stroke-width="3" opacity=".75"/>'
    '<circle cx="84" cy="28" r="9" fill="#fff3c4"/>'
    '<circle cx="101" cy="45" r="4.5" fill="#ffe08a"/>'
    '<path d="M24 34l2.6 6.6 6.6 2.6-6.6 2.6L24 52.4l-2.6-6.6-6.6-2.6 6.6-2.6z"'
    ' fill="#ffd977" opacity=".75"/>'
    '<path d="M48 14l1.8 4.6 4.6 1.8-4.6 1.8L48 26.8l-1.8-4.6-4.6-1.8 4.6-1.8z"'
    ' fill="#ffd977" opacity=".55"/>',
)

# ── Мифическая: алая рамка ──────────────────────────────────────────
AVATARS["rainbow"] = frame(
    "mythic",
    '<path d="M8 104a56 56 0 0 1 112 0h-15a41 41 0 0 0-82 0z" fill="#ef4444"/>'
    '<path d="M23 104a41 41 0 0 1 82 0H90a26 26 0 0 0-52 0z" fill="#f59e0b"/>'
    '<path d="M38 104a26 26 0 0 1 52 0H75a11 11 0 0 0-22 0z" fill="#22c55e"/>'
    '<path d="M53 104a11 11 0 0 1 22 0z" fill="#3b82f6"/>'
    '<circle cx="22" cy="106" r="11" fill="#fff" opacity=".9"/>'
    '<circle cx="31" cy="102" r="8" fill="#fff" opacity=".9"/>'
    '<circle cx="106" cy="106" r="11" fill="#fff" opacity=".9"/>'
    '<circle cx="97" cy="102" r="8" fill="#fff" opacity=".9"/>'
    '<path d="M96 20l3.4 8.6 8.6 3.4-8.6 3.4L96 44l-3.4-8.6-8.6-3.4 8.6-3.4z"'
    ' fill="#fff" opacity=".75"/>'
    '<path d="M34 30l2.4 6 6 2.4-6 2.4L34 47l-2.4-6.2-6-2.4 6-2.4z"'
    ' fill="#fff" opacity=".55"/>',
)


def main():
    os.makedirs(OUT, exist_ok=True)
    for name, svg in AVATARS.items():
        with open(os.path.join(OUT, name + ".svg"), "w", encoding="utf-8") as f:
            f.write(svg)
    print("аватарок записано:", len(AVATARS))


if __name__ == "__main__":
    main()
