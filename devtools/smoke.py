# -*- coding: utf-8 -*-
"""Прогон страниц в headless-браузере: ищем ошибки JS."""

import sys
from playwright.sync_api import sync_playwright

PAGES = [
    'assistant.html', 'profile.html', 'leaderboard.html', 'shop.html',
    'tests.html', 'authors.html', 'admin.html', 'lesson.html?id=fon-zvuki',
    'games.html', 'tree.html', 'class.html',
]
IGNORE = ('ERR_TUNNEL', 'Failed to load resource', 'net::')

SEED = """() => {
    localStorage.setItem('token', 't');
    localStorage.setItem('currentUser', JSON.stringify({
        username: 'demo', nickname: 'Демо', balance: 1250, xp: 640, streak: 5,
        badges: ['book', 'fire', 'trophy', 'rank_b1', '\\u{1F333}'],
        avatars: ['cat', 'dog', 'fox', 'dragon', 'crown', '\\u{1F98A}'],
        active_avatar: 'dragon', completed_topics: [], favorite_games: [],
        is_admin: true, research_consent: true
    }));
}"""

bad = 0
with sync_playwright() as p:
    b = p.chromium.launch()
    pg = b.new_context(viewport={'width': 1280, 'height': 900}).new_page()
    pg.goto('http://127.0.0.1:8099/index.html')
    pg.evaluate(SEED)
    for f in PAGES:
        errs = []
        h1 = lambda e: errs.append('JS: ' + str(e))
        h2 = lambda m: (errs.append('CONSOLE: ' + m.text)
                        if m.type == 'error' and not any(k in m.text for k in IGNORE)
                        else None)
        pg.on('pageerror', h1)
        pg.on('console', h2)
        pg.goto('http://127.0.0.1:8099/' + f)
        pg.wait_for_timeout(1500)
        pg.screenshot(path='/tmp/shot_' + f.split('.')[0] + '.png')
        pg.remove_listener('pageerror', h1)
        pg.remove_listener('console', h2)
        if errs:
            bad += 1
        print(('OK    ' if not errs else 'ОШИБКА'), f, errs[:3])
    b.close()

sys.exit(1 if bad else 0)
