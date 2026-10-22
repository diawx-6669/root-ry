# -*- coding: utf-8 -*-
"""Полный проход теста КСПОЯ в браузере: старт → ответы → результат."""

from playwright.sync_api import sync_playwright

SEED = """() => {
    localStorage.setItem('token', 't');
    localStorage.setItem('currentUser', JSON.stringify({
        username: 'demo', nickname: 'Демо', balance: 1250, xp: 640, streak: 5,
        badges: ['book', 'fire'], avatars: ['cat', 'dragon'],
        active_avatar: 'dragon', completed_topics: [], favorite_games: [],
        is_admin: false, research_consent: true
    }));
}"""

errs = []
with sync_playwright() as p:
    b = p.chromium.launch()
    pg = b.new_context(viewport={'width': 1280, 'height': 900}).new_page()
    pg.on('pageerror', lambda e: errs.append('JS: ' + str(e)))
    pg.on('console', lambda m: errs.append('CONSOLE: ' + m.text)
          if m.type == 'error' and 'net::' not in m.text
          and 'Failed to load resource' not in m.text else None)

    pg.goto('http://127.0.0.1:8099/index.html')
    pg.evaluate(SEED)
    pg.goto('http://127.0.0.1:8099/tests.html')
    pg.wait_for_timeout(1200)
    pg.screenshot(path='/tmp/t1_start.png')

    # Полноэкранный режим в headless не откроется — стартуем напрямую.
    pg.evaluate("() => startTest()")
    pg.wait_for_timeout(1200)
    pg.screenshot(path='/tmp/t2_question.png')

    # Отвечаем на все сорок вопросов первым вариантом.
    answered = pg.evaluate("""() => {
        for (let i = 0; i < 40; i++) {
            const opt = document.querySelector('.q-option');
            if (opt) opt.click();
            const next = document.getElementById('nextBtn');
            if (next && !next.disabled) next.click();
        }
        return currentQ;
    }""")
    pg.wait_for_timeout(400)
    pg.screenshot(path='/tmp/t3_last.png')

    # Момент нажатия «Завершить»: должен появиться экран проверки.
    pg.evaluate("() => finishTest(false)")
    pg.wait_for_timeout(120)
    checking = pg.evaluate(
        "() => document.getElementById('checkingScreen').classList.contains('active')")
    pg.screenshot(path='/tmp/t4_checking.png')

    pg.wait_for_timeout(1800)
    pg.screenshot(path='/tmp/t5_result.png')
    shown = pg.evaluate("""() => {
        const img = document.querySelector('.hero-badge img');
        const chip = document.querySelector('.rewards-row .reward-chip');
        return {
            heroSrc: img ? img.getAttribute('src') : null,
            heroLoaded: img ? img.naturalWidth > 0 : false,
            chipText: chip ? chip.textContent.trim() : null,
            chipImg: chip && chip.querySelector('img')
                     ? chip.querySelector('img').getAttribute('src') : null,
            resultVisible: document.getElementById('resultScreen')
                             .classList.contains('active'),
        };
    }""")
    b.close()

print('дошли до вопроса №', answered)
print('экран проверки показался:', checking)
print('результат:', shown)
print('ошибки JS:', errs or 'нет')
