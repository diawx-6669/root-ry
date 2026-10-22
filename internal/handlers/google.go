package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"rootry/internal/middleware"
	"rootry/internal/models"
)

// ═══════════════════════════════════════════════════════════════════
//  Вход и регистрация через Google.
//
//  Кнопку рисует библиотека Google Identity Services прямо на странице
//  входа. После выбора аккаунта браузер получает ID-токен (JWT,
//  подписанный Google) и отправляет его сюда. Сервер проверяет токен
//  у самого Google, и только после этого выдаёт свой токен RootRy.
//
//  Верить содержимому токена без проверки нельзя: подделать JWT с
//  чужой почтой может любой. Поэтому проверяется подпись (её проверяет
//  tokeninfo), получатель (aud — наш Client ID: токен, выданный другому
//  сайту, не подходит) и подтверждённость почты.
//
//  Первый вход создаёт аккаунт, повторный — входит в него. Отдельной
//  «регистрации через Google» нет: для ученика это одна кнопка.
// ═══════════════════════════════════════════════════════════════════

const googleTimeout = 10 * time.Second

// googleTokenInfoURL вынесен в переменную ради тестов.
var googleTokenInfoURL = "https://oauth2.googleapis.com/tokeninfo"

var googleIssuers = map[string]bool{
	"accounts.google.com":         true,
	"https://accounts.google.com": true,
}

type googleIdentity struct {
	Sub   string
	Email string
	Name  string
}

type googleAuthRequest struct {
	Credential string `json:"credential"`
}

// defaultGoogleClientID — Client ID прода (root-ry.onrender.com). Берётся,
// когда GOOGLE_CLIENT_ID не задан, чтобы вход через Google работал без
// ручной настройки Render. Для локальной разработки или другого домена
// переменная его перекрывает.
const defaultGoogleClientID = "601052756488-02htlat2dlmkvkrg3uoie7ohullv3cht.apps.googleusercontent.com"

func googleClientID() string {
	if id := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")); id != "" {
		return id
	}
	return defaultGoogleClientID
}

// GET /api/auth/google/config — Client ID для кнопки на странице входа.
//
// Client ID не секрет (он и так виден в коде любой страницы с кнопкой
// Google), но держать его в переменной окружения удобнее, чем вшивать
// в HTML: у локальной разработки и прода он может быть разный.
func (h *Handler) GoogleConfig(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"clientId": googleClientID()})
}

// POST /api/auth/google — вход или регистрация по ID-токену Google.
func (h *Handler) GoogleAuth(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	clientID := googleClientID()
	if clientID == "" {
		writeError(w, http.StatusServiceUnavailable, "Вход через Google не настроен")
		return
	}
	var req googleAuthRequest
	if err := h.parseBody(r, &req); err != nil || req.Credential == "" {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), googleTimeout)
	defer cancel()
	id, err := verifyGoogleToken(ctx, req.Credential, clientID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Google не подтвердил вход, попробуйте ещё раз")
		return
	}

	status := http.StatusOK
	user, ok := h.store.GetUserByGoogleSub(id.Sub)
	if !ok {
		user, err = h.store.CreateGoogleUser(id.Sub, id.Email, id.Name)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Не удалось создать аккаунт, попробуйте позже")
			return
		}
		status = http.StatusCreated
	} else if streak, updated := h.store.TouchDailyLogin(user.Username); updated {
		user.Streak = streak
	}

	token, _ := middleware.GenerateToken(user.ID, user.Username, user.IsAdmin)
	writeJSON(w, status, models.LoginResponse{Token: token, User: selfView(user)})
}

// verifyGoogleToken проверяет ID-токен через tokeninfo: Google сам
// сверяет подпись и срок действия, а мы — получателя, издателя и почту.
func verifyGoogleToken(ctx context.Context, credential, clientID string) (*googleIdentity, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		googleTokenInfoURL+"?id_token="+url.QueryEscape(credential), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tokeninfo: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tokeninfo: status %d", resp.StatusCode)
	}

	// tokeninfo отдаёт все поля строками, в том числе email_verified.
	var info struct {
		Aud           string `json:"aud"`
		Iss           string `json:"iss"`
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified string `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(nil, resp.Body, maxBodyBytes)).Decode(&info); err != nil {
		return nil, fmt.Errorf("tokeninfo: %w", err)
	}
	switch {
	case info.Aud != clientID:
		return nil, errors.New("token issued for another client")
	case !googleIssuers[info.Iss]:
		return nil, errors.New("unexpected issuer")
	case info.Sub == "":
		return nil, errors.New("no subject")
	case info.EmailVerified != "true":
		return nil, errors.New("email not verified")
	}
	return &googleIdentity{Sub: info.Sub, Email: info.Email, Name: info.Name}, nil
}
