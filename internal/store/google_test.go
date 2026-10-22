package store

import "testing"

func TestUsernameFromEmail(t *testing.T) {
	cases := map[string]string{
		"Ivan.Petrov-2010@gmail.com":      "ivan_petrov_2010",
		"a@gmail.com":                     "usera",
		"очень.русский@gmail.com":         "user",
		"averyveryverylongname1234@x.com": "averyveryverylon",
		"__x.y__@gmail.com":               "x_y",
	}
	for in, want := range cases {
		if got := usernameFromEmail(in); got != want {
			t.Errorf("usernameFromEmail(%q) = %q, ожидалось %q", in, got, want)
		}
		if got := withSuffix(usernameFromEmail(in)); len(got) > 20 {
			t.Errorf("логин с суффиксом длиннее 20 символов: %q", got)
		}
	}
}

// Первый вход через Google создаёт ученика, повторный возвращает того же.
// Занятый логин не мешает: к нему добавляется суффикс.
func TestCreateGoogleUser(t *testing.T) {
	s := testStore(t)
	cleanup := func(id int64) {
		t.Cleanup(func() { _, _ = s.db.Exec(`DELETE FROM users WHERE id=$1`, id) })
	}

	taken := testUser(t, s)
	var takenName string
	s.db.QueryRow(`SELECT username FROM users WHERE id=$1`, taken).Scan(&takenName)

	sub := "g_" + takenName
	u, err := s.CreateGoogleUser(sub, takenName+"@gmail.com", "Аня Иванова")
	if err != nil {
		t.Fatalf("не создать ученика: %v", err)
	}
	cleanup(u.ID)
	if u.Username == takenName || u.Nickname != "Аня Иванова" {
		t.Fatalf("неожиданный ученик: %q / %q", u.Username, u.Nickname)
	}

	again, ok := s.GetUserByGoogleSub(sub)
	if !ok || again.ID != u.ID {
		t.Fatalf("повторный вход нашёл не того ученика: %+v", again)
	}
	if dup, err := s.CreateGoogleUser(sub, "x@gmail.com", "X"); err != nil || dup.ID != u.ID {
		t.Fatalf("повторная регистрация создала второй аккаунт: %+v, %v", dup, err)
	}
}
