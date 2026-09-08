# -*- coding: utf-8 -*-
"""Заглушка API для проверки вёрстки без базы и без ключа модели."""

import json, os, http.server, socketserver

ROOT = os.path.join(os.path.dirname(__file__), "..", "static")

USER = {
    "id": 1, "username": "demo", "nickname": "Демо", "balance": 1250, "xp": 640,
    "streak": 5, "badges": ["book", "fire", "trophy", "rank_b1", "🌳"],
    "avatars": ["cat", "dog", "fox", "dragon", "crown", "🦊"],
    "active_avatar": "dragon", "completed_topics": ["fon-zvuki"],
    "favorite_games": [], "games_won_types": [], "promo_used": [],
    "is_admin": True, "research_consent": True, "last_daily_claim": "",
    "daily_tasks_date": "", "games_won_today": 2,
}

PLAN = {
    "due": [
        {"topic_id": "fon-udarenie", "title": "Ударение", "section": "1. ФОНЕТИКА",
         "box": 2, "overdue_days": 3, "xp": 30, "coins": 15},
        {"topic_id": "orf-n-nn", "title": "Н и НН", "section": "3. ОРФОГРАФИЯ",
         "box": 1, "overdue_days": 0, "xp": 40, "coins": 20},
    ],
    "mastered": 4, "learned": 11, "total_topics": 74,
}

MISTAKES = {
    "days_to_resolve": 2,
    "items": [
        {"topic_id": "orf-n-nn", "item_id": "orf-n-nn#2", "wrong_count": 3,
         "last_wrong": "2026-09-05"},
        {"topic_id": "fon-udarenie", "item_id": "fon-udarenie#0", "wrong_count": 1,
         "last_wrong": "2026-09-07"},
    ],
}

CHAT = {
    "reply": "Привет! Рад познакомиться. Расскажи, что тебе больше всего "
             "нравится в школе — какой предмет ждёшь каждую неделю?",
    "mistakes": [
        {"wrong": "я пошёл в школа", "right": "я пошёл в школу",
         "why": "После глагола движения нужен винительный падеж: пошёл (куда?) в школу.",
         "kind": "грамматика"},
        {"wrong": "ево", "right": "его",
         "why": "В местоимении «его» пишется буква «г», хотя произносится [в].",
         "kind": "орфография"},
    ],
    "note": "Хорошо построил предложение с придаточным — это сложная конструкция.",
}

ROUTES = {
    "/api/me": USER,
    "/api/review/plan": PLAN,
    "/api/mistakes": MISTAKES,
    "/api/kspoya/status": {"best": {"score": 27, "level": "B1"}},
    "/api/assistant/chat": CHAT,
    "/api/admin/stats": {"active_today": 12, "total_users": 84, "total_games": 431,
                         "total_cases": 96, "total_xp": 51240, "total_balance": 78300,
                         "total_badges": 61, "total_avatars": 143},
    "/api/admin/users": {"users": [dict(USER, level=4)]},
    "/api/case/open": {"item_emoji": "dragon", "item_rarity": "rare",
                       "is_duplicate": False, "compensation": 0,
                       "new_balance": 650, "new_xp": 700},
    "/api/leaderboard": [
        dict(username="demo", nickname="Демо", xp=640, balance=1250, badges=5,
             badges_count=5, streak=5, active_avatar="dragon",
             kspoya_score=27, kspoya_level="B1"),
        dict(username="ann", nickname="Аня", xp=980, balance=430, badges=3,
             badges_count=3, streak=12, active_avatar="fox",
             kspoya_score=31, kspoya_level="B2"),
        dict(username="tim", nickname="Тимур", xp=310, balance=90, badges=1,
             badges_count=1, streak=1, active_avatar="frog",
             kspoya_score=0, kspoya_level=""),
    ],
}


class H(http.server.SimpleHTTPRequestHandler):
    def __init__(self, *a, **kw):
        super().__init__(*a, directory=os.path.abspath(ROOT), **kw)

    def _api(self):
        path = self.path.split("?")[0]
        if path in ROUTES:
            body = json.dumps(ROUTES[path], ensure_ascii=False).encode()
            self.send_response(200)
            self.send_header("Content-Type", "application/json; charset=utf-8")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return True
        return False

    def do_GET(self):
        if self.path.startswith("/api/"):
            if not self._api():
                self.send_error(404)
            return
        super().do_GET()

    def do_POST(self):
        length = int(self.headers.get("Content-Length") or 0)
        self.rfile.read(length)
        if not self._api():
            self.send_error(404)

    do_PUT = do_POST

    def log_message(self, *a):
        pass


if __name__ == "__main__":
    socketserver.TCPServer.allow_reuse_address = True
    with socketserver.TCPServer(("127.0.0.1", 8099), H) as srv:
        srv.serve_forever()
