package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Токен Google принимается, только если он выдан нашему Client ID,
// издан Google и почта подтверждена. Всё остальное — отказ.
func TestVerifyGoogleToken(t *testing.T) {
	const clientID = "rootry.apps.googleusercontent.com"
	cases := []struct {
		name   string
		status int
		body   string
		ok     bool
	}{
		{"верный токен", 200,
			`{"aud":"` + clientID + `","iss":"https://accounts.google.com","sub":"42","email":"a@b.c","email_verified":"true","name":"Аня"}`, true},
		{"чужой Client ID", 200,
			`{"aud":"other.apps.googleusercontent.com","iss":"accounts.google.com","sub":"42","email_verified":"true"}`, false},
		{"чужой издатель", 200,
			`{"aud":"` + clientID + `","iss":"evil.example","sub":"42","email_verified":"true"}`, false},
		{"почта не подтверждена", 200,
			`{"aud":"` + clientID + `","iss":"accounts.google.com","sub":"42","email_verified":"false"}`, false},
		{"Google отклонил токен", 400, `{"error":"invalid_token"}`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("id_token") != "tok" {
					t.Errorf("id_token не передан: %q", r.URL.RawQuery)
				}
				w.WriteHeader(c.status)
				w.Write([]byte(c.body))
			}))
			defer srv.Close()
			old := googleTokenInfoURL
			googleTokenInfoURL = srv.URL
			defer func() { googleTokenInfoURL = old }()

			id, err := verifyGoogleToken(context.Background(), "tok", clientID)
			if c.ok && (err != nil || id.Sub != "42" || id.Name != "Аня") {
				t.Fatalf("ожидался успех, получено %+v, %v", id, err)
			}
			if !c.ok && err == nil {
				t.Fatal("токен принят, а должен быть отклонён")
			}
		})
	}
}
