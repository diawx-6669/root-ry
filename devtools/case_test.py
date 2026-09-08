# -*- coding: utf-8 -*-
"""Проверка рулетки магазина после перехода на SVG."""

from playwright.sync_api import sync_playwright

SEED = """() => {
    localStorage.setItem('token', 't');
    localStorage.setItem('currentUser', JSON.stringify({
        username: 'demo', nickname: 'Демо', balance: 5000,
        active_avatar: 'cat', avatars: ['cat'], badges: [], streak: 3
    }));
}"""

with sync_playwright() as p:
    b = p.chromium.launch()
    pg = b.new_context(viewport={'width': 1280, 'height': 900}).new_page()
    errs = []
    pg.on('pageerror', lambda e: errs.append(str(e)))
    pg.goto('http://127.0.0.1:8099/index.html')
    pg.evaluate(SEED)
    pg.goto('http://127.0.0.1:8099/shop.html')
    pg.wait_for_timeout(1200)
    pg.evaluate("() => openCase('rare')")
    pg.wait_for_timeout(1200)
    pg.screenshot(path='/tmp/shot_case1.png')
    pg.wait_for_timeout(7000)
    pg.screenshot(path='/tmp/shot_case2.png')
    print('ошибки JS:', errs)
    b.close()
