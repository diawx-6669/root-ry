/* ═══════════════════════════════════════════════════════════════════
   Иконки RootRy.

   Раньше всё мелкое оформление держалось на эмодзи: огонёк у серии,
   монета у баланса, медаль у значков. Эмодзи рисует шрифт операционной
   системы — на школьном компьютере, на телефоне и на проекторе актового
   зала один и тот же экран выглядел по-разному, а кое-где вместо
   картинки оставался пустой квадрат.

   Здесь те же смыслы заданы векторными путями: они одинаковы везде,
   красятся через currentColor и не требуют сети.
   ═══════════════════════════════════════════════════════════════════ */

const ICON_PATHS = {
    // Прогресс и награды
    fire:     '<path d="M12 2c5 5 8 8.5 8 13a8 8 0 0 1-16 0c0-3 1.5-5.3 3.7-7.4.7 2.3 1.9 3.5 3 3.9C9.3 7.9 10.6 4.8 12 2z"/><path d="M12 11.5c2.5 3 4 4.7 4 6.8a4 4 0 0 1-8 0c0-2.1 1.5-3.8 4-6.8z" fill="#fff" opacity=".45"/>',
    coin:     '<ellipse cx="12" cy="12" rx="9" ry="9"/><path d="M12 6.5c-2.2 0-3.8 1-3.8 2.6 0 3 6 1.9 6 3.9 0 .9-1 1.5-2.2 1.5-1.4 0-2.3-.6-2.5-1.6H7.9c.1 1.9 1.5 3 3.4 3.2V18h1.4v-1.9c2-.2 3.4-1.3 3.4-3 0-3.2-6-2.1-6-4 0-.8.9-1.3 2.1-1.3 1.2 0 2.1.6 2.2 1.5h1.6c-.1-1.7-1.4-2.7-3.2-2.9V6h-1.4z" fill="#fff" opacity=".85"/>',
    bolt:     '<path d="M13.2 2 4 13.4h6.1L9.9 22 20 10.2h-6.4z"/>',
    star:     '<path d="m12 2.6 2.9 6.2 6.6.8-4.9 4.6 1.3 6.7L12 17.5l-5.9 3.4 1.3-6.7-4.9-4.6 6.6-.8z"/>',
    trophy:   '<path d="M7 3h10v5a5 5 0 0 1-10 0z"/><path d="M7 4.5H4.5A3.5 3.5 0 0 0 8 9M17 4.5h2.5A3.5 3.5 0 0 1 16 9" fill="none" stroke="currentColor" stroke-width="1.7"/><path d="M10.7 12.5h2.6V16h-2.6z"/><rect x="7" y="16" width="10" height="3" rx="1.3"/>',
    medal:    '<path d="M7 2h3l2.4 6H9.4zM14 2h3l-2.4 8h-3z"/><circle cx="12" cy="15.5" r="6.2"/><circle cx="12" cy="15.5" r="3.4" fill="#fff" opacity=".55"/>',
    gem:      '<path d="M6 3h12l4 5-10 13L2 8z"/><path d="M6 3l2 5h8l2-5M2 8h20M12 21 8 8M12 21l4-13" fill="none" stroke="#fff" stroke-width="1.2" opacity=".5"/>',
    crown:    '<path d="M3 18 2 5l6 4 4-6.5L16 9l6-4-1 13z"/><rect x="3" y="18" width="18" height="3" rx="1.2"/>',
    target:   '<circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" stroke-width="2"/><circle cx="12" cy="12" r="5" fill="none" stroke="currentColor" stroke-width="2"/><circle cx="12" cy="12" r="1.8"/>',
    chart:    '<rect x="3" y="12" width="4" height="9" rx="1.3"/><rect x="10" y="7" width="4" height="14" rx="1.3"/><rect x="17" y="3" width="4" height="18" rx="1.3"/>',
    bulb:     '<path d="M12 2a7 7 0 0 1 4 12.7V17H8v-2.3A7 7 0 0 1 12 2z"/><rect x="9" y="18" width="6" height="2" rx="1"/><rect x="10" y="21" width="4" height="1.8" rx=".9"/>',

    // Учёба
    book:     '<path d="M3 4h6a3 3 0 0 1 3 2 3 3 0 0 1 3-2h6v14h-6a3 3 0 0 0-3 2 3 3 0 0 0-3-2H3z"/><path d="M12 6.5v14" fill="none" stroke="#fff" stroke-width="1.3" opacity=".6"/>',
    notepad:  '<rect x="4" y="2" width="16" height="20" rx="2.6"/><path d="M8 7h8M8 11h8M8 15h5" fill="none" stroke="#fff" stroke-width="1.7" stroke-linecap="round" opacity=".8"/>',
    pen:      '<path d="M15.5 3.3 20.7 8.5 9.2 20H4v-5.2z"/><path d="m15.5 3.3 5.2 5.2 1.5-1.5a3.7 3.7 0 0 0-5.2-5.2z" opacity=".6"/>',
    tree:     '<circle cx="12" cy="6" r="4.2"/><circle cx="6.6" cy="11.4" r="3.4"/><circle cx="17.4" cy="11.4" r="3.4"/><rect x="10.8" y="10" width="2.4" height="11" rx="1.1"/><path d="m12 15-3.2-2.6M12 18l3.2-2.6" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/>',
    grad:     '<path d="M12 3 1.5 8 12 13l10.5-5z"/><path d="M5 10.4V15c0 1.9 3.1 3.5 7 3.5s7-1.6 7-3.5v-4.6l-7 3.3z"/><path d="M21 9v6" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/>',
    flask:    '<path d="M9 2h6v6.2l4.6 8.9A3 3 0 0 1 16.9 22H7.1a3 3 0 0 1-2.7-4.9L9 8.2z" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linejoin="round"/><path d="M7.4 14.5h9.2" fill="none" stroke="currentColor" stroke-width="1.9"/>',

    // Статусы
    check:    '<path d="m4.5 12.4 5 5 10-10.8" fill="none" stroke="currentColor" stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round"/>',
    cross:    '<path d="M6 6l12 12M18 6 6 18" fill="none" stroke="currentColor" stroke-width="2.6" stroke-linecap="round"/>',
    warn:     '<path d="M12 2.5 22.5 21H1.5z"/><path d="M12 9v5.4M12 17.2v.1" fill="none" stroke="#fff" stroke-width="2.2" stroke-linecap="round"/>',
    lock:     '<rect x="4.5" y="10" width="15" height="11" rx="3"/><path d="M8 10V7.5a4 4 0 0 1 8 0V10" fill="none" stroke="currentColor" stroke-width="2.2"/>',
    ban:      '<circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" stroke-width="2.2"/><path d="m6 6 12 12" stroke="currentColor" stroke-width="2.2"/>',
    dot:      '<circle cx="12" cy="12" r="6"/>',
    heart:    '<path d="M12 21S3 14.6 3 8.9A5.1 5.1 0 0 1 12 5.7 5.1 5.1 0 0 1 21 8.9C21 14.6 12 21 12 21z"/>',
    question: '<circle cx="12" cy="12" r="9.2" fill="none" stroke="currentColor" stroke-width="2"/><path d="M9.3 9.2a2.8 2.8 0 0 1 5.4.9c0 1.9-2.7 2-2.7 4M12 17.4v.1" fill="none" stroke="currentColor" stroke-width="2.1" stroke-linecap="round"/>',

    // Интерфейс
    soundOn:  '<path d="M4 9h3.5L13 4.5v15L7.5 15H4z"/><path d="M16 9.2a4 4 0 0 1 0 5.6M18.6 6.6a7.6 7.6 0 0 1 0 10.8" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round"/>',
    soundOff: '<path d="M4 9h3.5L13 4.5v15L7.5 15H4z"/><path d="m16.5 9.5 5 5M21.5 9.5l-5 5" fill="none" stroke="currentColor" stroke-width="2.1" stroke-linecap="round"/>',
    gamepad:  '<rect x="1.5" y="7" width="21" height="11" rx="5.5"/><path d="M7 10v4M5 12h4M16 11.2v.1M18.4 13.4v.1" fill="none" stroke="#fff" stroke-width="2.1" stroke-linecap="round"/>',
    box:      '<path d="M3 7.5 12 3l9 4.5V17L12 21.5 3 17z"/><path d="m3 7.5 9 4.5 9-4.5M12 12v9.5" fill="none" stroke="#fff" stroke-width="1.4" opacity=".55"/>',
    users:    '<circle cx="9" cy="8" r="3.8"/><path d="M2.5 20a6.5 6.5 0 0 1 13 0z"/><circle cx="17.5" cy="9" r="3"/><path d="M14 20h7.5a5 5 0 0 0-6.2-4.8" opacity=".65"/>',
    palette:  '<path d="M12 2.6a9.4 9.4 0 0 0 0 18.8c1.4 0 2.2-.9 2.2-2 0-1.6 1-2.2 2.4-2.2h1.6a4.4 4.4 0 0 0 4.4-4.4C22.6 6.4 18 2.6 12 2.6z" fill="none" stroke="currentColor" stroke-width="1.9"/><circle cx="7.5" cy="11" r="1.5"/><circle cx="10.5" cy="7" r="1.5"/><circle cx="15" cy="7.5" r="1.5"/><circle cx="17.5" cy="11.5" r="1.5"/>',
    robot:    '<rect x="3.5" y="7" width="17" height="13" rx="4"/><path d="M12 3v4M9.5 13v.1M14.5 13v.1M9 16.6h6" fill="none" stroke="#fff" stroke-width="2.2" stroke-linecap="round"/><circle cx="12" cy="2.6" r="1.6"/>',
    send:     '<path d="M3 11.5 21 3l-8.5 18-2.3-7.2z"/>',
    spark:    '<path d="m12 2 2 6 6 2-6 2-2 6-2-6-6-2 6-2z"/><path d="m19 14 1 2.6 2.6 1-2.6 1-1 2.6-1-2.6-2.6-1 2.6-1z" opacity=".65"/>',
    calendar: '<rect x="3" y="5" width="18" height="16" rx="3"/><path d="M3 10h18M8 3v4M16 3v4" fill="none" stroke="#fff" stroke-width="1.9" stroke-linecap="round"/>',
    refresh:  '<path d="M20 12a8 8 0 1 1-2.6-5.9" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round"/><path d="M20 3v5h-5"  fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"/>',
};

/**
 * Разметка иконки.
 *
 * @param {string} id      ключ из ICON_PATHS
 * @param {object} [opts]  { size, cls, color, title }
 */
function svgIcon(id, opts) {
    const o = opts || {};
    const body = ICON_PATHS[id];
    if (!body) return '';
    const size = o.size || 18;
    const cls = o.cls ? ` class="${o.cls}"` : '';
    const style = 'vertical-align:-.15em;flex:none' + (o.color ? `;color:${o.color}` : '');
    const title = o.title ? `<title>${o.title}</title>` : '';
    const label = o.title ? '' : ' aria-hidden="true"';
    return (
        `<svg${cls} width="${size}" height="${size}" viewBox="0 0 24 24" ` +
        `fill="currentColor" focusable="false"${label} style="${style}">` +
        title + body + '</svg>'
    );
}

/** Иконка как отдельный DOM-узел — для мест, где нужен element, а не строка. */
function svgIconEl(id, opts) {
    const wrap = document.createElement('span');
    wrap.style.display = 'inline-flex';
    wrap.innerHTML = svgIcon(id, opts);
    return wrap;
}
