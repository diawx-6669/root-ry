# -*- coding: utf-8 -*-
"""
Обход сайта: на каждой странице жмём всё, что жмётся, и собираем ошибки JS.

Заглушка API (mockserver.py) должна быть запущена.
"""

import sys
from playwright.sync_api import sync_playwright

PAGES = [
    'games.html', 'tree.html', 'tests.html', 'assistant.html', 'shop.html',
    'leaderboard.html', 'profile.html', 'authors.html', 'class.html',
    'admin.html', 'lesson.html?id=fon-zvuki',
]

# Клики по этим селекторам могут увести со страницы или открыть диалог.
SKIP = ('logout', 'API.logout', 'startTest', 'openCase', 'claimDaily')

IGNORE = ('net::', 'Failed to load resource', 'ERR_TUNNEL')

SEED = """() => {
    localStorage.setItem('token', 't');
    localStorage.setItem('currentUser', JSON.stringify({
        username: 'demo', nickname: 'Демо', balance: 5000, xp: 640, streak: 5,
        badges: ['book', 'fire', 'trophy', 'rank_b1'],
        avatars: ['cat', 'corgi', 'arcticfox', 'dragon'],
        active_avatar: 'dragon', completed_topics: ['fon-zvuki'],
        favorite_games: [], games_won_types: [], promo_used: [],
        is_admin: true, is_teacher: true, research_consent: true
    }));
}"""

found = 0
with sync_playwright() as p:
    b = p.chromium.launch()
    pg = b.new_context(viewport={'width': 1280, 'height': 900}).new_page()
    pg.goto('http://127.0.0.1:8099/index.html')
    pg.evaluate(SEED)

    for page in PAGES:
        errs = []
        h1 = lambda e: errs.append('JS: ' + str(e))
        h2 = lambda m: (errs.append('CONSOLE: ' + m.text)
                        if m.type == 'error' and not any(k in m.text for k in IGNORE)
                        else None)
        pg.on('pageerror', h1)
        pg.on('console', h2)

        pg.goto('http://127.0.0.1:8099/' + page)
        pg.wait_for_timeout(1200)

        # Собираем кликабельное, кроме навигации и опасных действий.
        targets = pg.evaluate("""(skip) => {
            const out = [];
            document.querySelectorAll('button, .tab-btn, .tab, [onclick]').forEach((el, i) => {
                const code = (el.getAttribute('onclick') || '') + ' ' + (el.className || '');
                if (skip.some(s => code.includes(s))) return;
                if (el.closest('.bottom-tabbar')) return;
                el.setAttribute('data-crawl', 'c' + i);
                out.push('c' + i);
            });
            return out;
        }""", list(SKIP))

        for t in targets[:24]:
            try:
                el = pg.query_selector(f'[data-crawl="{t}"]')
                if el and el.is_visible():
                    el.click(timeout=1200)
                    pg.wait_for_timeout(220)
            except Exception:
                pass  # невидимое или перекрытое — не ошибка страницы

        pg.remove_listener('pageerror', h1)
        pg.remove_listener('console', h2)
        uniq = sorted(set(errs))
        if uniq:
            found += 1
            print('ОШИБКА', page)
            for e in uniq[:6]:
                print('   ', e)
        else:
            print('OK    ', page, f'(нажато {len(targets[:24])})')
    b.close()

sys.exit(1 if found else 0)
