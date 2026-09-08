# -*- coding: utf-8 -*-
"""
Палитра редкостей — общая для аватарок и значков.

Редкость видна раньше, чем сам предмет: по фону. Раньше фон задавался
зверем (кот — голубой, лев — жёлтый), и легендарная аватарка выглядела
ровно так же весомо, как обычная. Теперь тон фона и цвет ободка идут от
редкости, а различает предметы сама фигура.

Цвета обязаны совпадать с RARITY_COLORS в static/shop.html.
"""

# ключ → (внутренний тон фона, внешний тон фона, ободок, акцент подписи)
RARITY = {
    "common":    ("#f3f5f7", "#dde3e9", "#9aa7b4", "#6b7280"),
    "rare":      ("#eaf3ff", "#c6dcfb", "#5b9bf8", "#3b82f6"),
    "epic":      ("#f6edff", "#dfcbfb", "#b072f5", "#a855f7"),
    "legendary": ("#fff8e6", "#ffe3a6", "#f0a91f", "#f59e0b"),
    "mythic":    ("#ffeee9", "#ffd0c4", "#f4664f", "#ef4444"),
}

ORDER = ["common", "rare", "epic", "legendary", "mythic"]


def gradient(rarity, gid, size=128):
    """Радиальный фон редкости: светлее к центру, плотнее к краям."""
    inner, outer, _, _ = RARITY[rarity]
    return (
        f'<radialGradient id="{gid}" cx="50%" cy="38%" r="78%">'
        f'<stop offset="0%" stop-color="{inner}"/>'
        f'<stop offset="100%" stop-color="{outer}"/>'
        "</radialGradient>"
    )


def ring(rarity):
    return RARITY[rarity][2]


def accent(rarity):
    return RARITY[rarity][3]
