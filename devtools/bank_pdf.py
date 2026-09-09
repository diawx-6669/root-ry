#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Сборка PDF со всем банком вопросов КСПОЯ — приложение к защите.

Источник данных — сам банк, а не отдельная копия: JSON получается
командой `go run ./devtools/dumpbank`. Так документ не может разойтись
с тем, что видит ученик на сайте.

    go run ./devtools/dumpbank > bank.json
    python3 devtools/bank_pdf.py bank.json КСПОЯ-банк-вопросов.pdf
"""

import json
import sys
from collections import Counter, OrderedDict

from reportlab.lib import colors
from reportlab.lib.enums import TA_CENTER, TA_JUSTIFY
from reportlab.lib.pagesizes import A4
from reportlab.lib.styles import ParagraphStyle
from reportlab.lib.units import mm
from reportlab.pdfbase import pdfmetrics
from reportlab.pdfbase.ttfonts import TTFont
from reportlab.platypus import (
    BaseDocTemplate, Frame, KeepTogether, PageBreak, PageTemplate,
    Paragraph, Spacer, Table, TableStyle,
)

# ── Шрифт ───────────────────────────────────────────────────────────
# Liberation Serif метрически совпадает с Times New Roman и содержит
# кириллицу. Встроенные шрифты reportlab кириллицу не умеют вовсе.
FONT_DIR = "/usr/share/fonts/truetype/liberation"
pdfmetrics.registerFont(TTFont("TNR", f"{FONT_DIR}/LiberationSerif-Regular.ttf"))
pdfmetrics.registerFont(TTFont("TNR-Bold", f"{FONT_DIR}/LiberationSerif-Bold.ttf"))
pdfmetrics.registerFont(TTFont("TNR-Italic", f"{FONT_DIR}/LiberationSerif-Italic.ttf"))
pdfmetrics.registerFontFamily("TNR", normal="TNR", bold="TNR-Bold", italic="TNR-Italic")

# Галочка. В Liberation Serif глифа U+2713 нет вовсе, и на его месте
# в PDF оставалась пустота — правильный ответ помечался только заливкой.
pdfmetrics.registerFont(TTFont("Sym", "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"))

INK = colors.HexColor("#10222c")
MUTED = colors.HexColor("#5b7280")
ACCENT = colors.HexColor("#0099cc")
RIGHT_BG = colors.HexColor("#e8f4ef")
RIGHT_INK = colors.HexColor("#14624a")
RULE = colors.HexColor("#d7e2e8")

LEVEL_TITLES = OrderedDict([
    ("A1", "A1 — Начальный"),
    ("A2", "A2 — Элементарный"),
    ("B1", "B1 — Средний"),
    ("B2", "B2 — Выше среднего"),
    ("C1", "C1 — Продвинутый"),
    ("C2", "C2 — Мастерство"),
])

S = {
    "title": ParagraphStyle("title", fontName="TNR-Bold", fontSize=22, leading=27,
                            alignment=TA_CENTER, textColor=INK, spaceAfter=6),
    "subtitle": ParagraphStyle("subtitle", fontName="TNR", fontSize=13, leading=18,
                               alignment=TA_CENTER, textColor=MUTED, spaceAfter=18),
    "lead": ParagraphStyle("lead", fontName="TNR", fontSize=12, leading=18,
                           alignment=TA_JUSTIFY, textColor=INK, spaceAfter=8),
    "h2": ParagraphStyle("h2", fontName="TNR-Bold", fontSize=15, leading=20,
                         textColor=INK, spaceBefore=6, spaceAfter=10),
    "qhead": ParagraphStyle("qhead", fontName="TNR", fontSize=9.5, leading=12,
                            textColor=MUTED, spaceAfter=2),
    "qtext": ParagraphStyle("qtext", fontName="TNR-Bold", fontSize=12, leading=16,
                            textColor=INK, spaceAfter=4),
    "opt": ParagraphStyle("opt", fontName="TNR", fontSize=11, leading=15,
                          textColor=INK, leftIndent=10),
    "optright": ParagraphStyle("optright", fontName="TNR-Bold", fontSize=11, leading=15,
                               textColor=RIGHT_INK, leftIndent=10),
    "explain": ParagraphStyle("explain", fontName="TNR-Italic", fontSize=10.5, leading=14,
                              textColor=MUTED, leftIndent=10, spaceBefore=3, spaceAfter=2),
    "cell": ParagraphStyle("cell", fontName="TNR", fontSize=11, leading=15, textColor=INK),
    "cellb": ParagraphStyle("cellb", fontName="TNR-Bold", fontSize=11, leading=15, textColor=INK),
}

LETTERS = "АБВГДЕЖЗИК"


def esc(text):
    return (str(text).replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;"))


def draw_page(canvas, doc):
    """Колонтитул: название слева, номер страницы справа."""
    canvas.saveState()
    canvas.setFont("TNR", 9)
    canvas.setFillColor(MUTED)
    canvas.drawString(20 * mm, 12 * mm, "RootRy · КСПОЯ · банк вопросов")
    canvas.drawRightString(A4[0] - 20 * mm, 12 * mm, str(canvas.getPageNumber()))
    canvas.setStrokeColor(RULE)
    canvas.setLineWidth(0.5)
    canvas.line(20 * mm, 16 * mm, A4[0] - 20 * mm, 16 * mm)
    canvas.restoreState()


def question_block(q, number):
    """Один вопрос: шапка, текст, варианты, правильный ответ, разбор."""
    flow = [
        Paragraph(f"Вопрос № {q['ID']} &nbsp;·&nbsp; {esc(q['Level'])} &nbsp;·&nbsp; {esc(q['Topic'])}",
                  S["qhead"]),
        Paragraph(f"{number}. {esc(q['Text'])}", S["qtext"]),
    ]

    rows = []
    for i, opt in enumerate(q["Options"]):
        right = (i == q["Correct"])
        letter = LETTERS[i] if i < len(LETTERS) else str(i + 1)
        mark = '<font name="Sym" color="#14624a">\u2713</font>' if right else ""
        rows.append([
            Paragraph(f"{letter})", S["optright"] if right else S["opt"]),
            Paragraph(esc(opt), S["optright"] if right else S["opt"]),
            Paragraph(mark, S["optright"]),
        ])

    table = Table(rows, colWidths=[12 * mm, 132 * mm, 8 * mm], hAlign="LEFT")
    style = [
        ("VALIGN", (0, 0), (-1, -1), "TOP"),
        ("TOPPADDING", (0, 0), (-1, -1), 2),
        ("BOTTOMPADDING", (0, 0), (-1, -1), 2),
        ("LEFTPADDING", (0, 0), (-1, -1), 3),
    ]
    style.append(("BACKGROUND", (0, q["Correct"]), (-1, q["Correct"]), RIGHT_BG))
    table.setStyle(TableStyle(style))
    flow.append(table)

    letter = LETTERS[q["Correct"]] if q["Correct"] < len(LETTERS) else str(q["Correct"] + 1)
    flow.append(Spacer(1, 3))
    flow.append(Paragraph(
        f'<b>Правильный ответ:</b> {letter}) {esc(q["Options"][q["Correct"]])}',
        S["cell"]))
    if q.get("Explain"):
        flow.append(Paragraph(f'<b>Разбор:</b> {esc(q["Explain"])}', S["explain"]))
    flow.append(Spacer(1, 9))
    return KeepTogether(flow)


def build(bank, out_path):
    doc = BaseDocTemplate(
        out_path, pagesize=A4,
        leftMargin=20 * mm, rightMargin=20 * mm,
        topMargin=18 * mm, bottomMargin=20 * mm,
        title="КСПОЯ — банк вопросов",
        author="RootRy",
        subject="Диагностический тест КСПОЯ: все вопросы, варианты и правильные ответы",
    )
    frame = Frame(doc.leftMargin, doc.bottomMargin, doc.width, doc.height, id="main")
    doc.addPageTemplates([PageTemplate(id="all", frames=[frame], onPage=draw_page)])

    story = []

    # ── Титул ──
    story.append(Spacer(1, 30 * mm))
    story.append(Paragraph("КСПОЯ", S["title"]))
    story.append(Paragraph(
        "Казахстанская система проверки и оценивания языка<br/>"
        "Банк вопросов диагностического теста", S["subtitle"]))

    by_level = Counter(q["Level"] for q in bank)
    facts = [
        ["Всего вопросов в банке", str(len(bank))],
        ["Вопросов в одной попытке", "40"],
        ["Вариантов ответа у вопроса", "6"],
        ["Время на тест", "40 минут"],
        ["Ступеней сложности", "6 (A1–C2)"],
        ["Вопросов на ступенях",
         " · ".join(f"{lvl} — {by_level[lvl]}" for lvl in LEVEL_TITLES if by_level[lvl])],
        ["Разделов грамматики", str(len({q["Topic"].split(" · ")[0] for q in bank}))],
    ]
    tbl = Table([[Paragraph(a, S["cell"]), Paragraph(b, S["cellb"])] for a, b in facts],
                colWidths=[85 * mm, 65 * mm], hAlign="CENTER")
    tbl.setStyle(TableStyle([
        ("VALIGN", (0, 0), (-1, -1), "MIDDLE"),
        ("TOPPADDING", (0, 0), (-1, -1), 5),
        ("BOTTOMPADDING", (0, 0), (-1, -1), 5),
        ("LINEBELOW", (0, 0), (-1, -2), 0.4, RULE),
        ("BOX", (0, 0), (-1, -1), 0.6, RULE),
        ("LEFTPADDING", (0, 0), (-1, -1), 8),
    ]))
    story.append(tbl)
    story.append(Spacer(1, 12 * mm))
    story.append(Paragraph(
        "Документ собран прямо из банка вопросов платформы, поэтому он не может "
        "разойтись с тем, что видит ученик. Правильный ответ выделен цветом и "
        "галочкой; под каждым вопросом приведён разбор — то самое правило, которое "
        "ученик получает на экране результата после сдачи теста.", S["lead"]))
    story.append(Paragraph(
        "На самом тесте порядок вариантов у каждого ученика свой: он зависит от "
        "номера попытки, поэтому буквы ответов здесь и на экране различаются, а "
        "стратегия «всегда жать один и тот же вариант» не работает.", S["lead"]))
    story.append(Paragraph(
        "Вопросов на ступенях неодинаково, и это сделано намеренно: в один тест "
        "с C1 попадает втрое больше вопросов, чем с A1, поэтому наверху банк должен "
        "быть глубже — иначе два прохода подряд повторялись бы именно там, где "
        "решается уровень.", S["lead"]))
    story.append(PageBreak())

    # ── Вопросы по ступеням ──
    number = 0
    for level, title in LEVEL_TITLES.items():
        items = [q for q in bank if q["Level"] == level]
        if not items:
            continue
        story.append(Paragraph(f"Ступень {title} — {len(items)} вопросов", S["h2"]))
        for q in items:
            number += 1
            story.append(question_block(q, number))
        story.append(PageBreak())

    if story and isinstance(story[-1], PageBreak):
        story.pop()

    doc.build(story)


def main():
    src = sys.argv[1] if len(sys.argv) > 1 else "bank.json"
    out = sys.argv[2] if len(sys.argv) > 2 else "КСПОЯ-банк-вопросов.pdf"
    with open(src, encoding="utf-8") as fh:
        bank = json.load(fh)
    build(bank, out)
    print(f"{out}: {len(bank)} вопросов")


if __name__ == "__main__":
    main()
