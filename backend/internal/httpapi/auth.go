package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"clyra/backend/internal/database"
	"clyra/backend/internal/password"
	"clyra/backend/internal/problem"
	"clyra/backend/internal/textutil"
)

const sessionCookieName = "clyra_session"
const sessionLifetime = 7 * 24 * time.Hour

type publicUser struct {
	ID        string  `json:"id"`
	Username  string  `json:"username"`
	Nickname  *string `json:"nickname"`
	AvatarURL *string `json:"avatar_url"`
}
type adminRecord struct {
	publicUser
	PasswordHash   string
	SessionVersion int64
}
type adminContextKey struct{}

func currentAdmin(r *http.Request) adminRecord {
	return r.Context().Value(adminContextKey{}).(adminRecord)
}
func (a *API) readAdmin() (adminRecord, error) {
	var admin adminRecord
	err := a.db.QueryRow(`SELECT id,username,nickname,avatar_url,password_hash,session_version FROM users LIMIT 1`).Scan(&admin.ID, &admin.Username, &admin.Nickname, &admin.AvatarURL, &admin.PasswordHash, &admin.SessionVersion)
	return admin, err
}

type sessionClaims struct {
	ID      string `json:"id"`
	AdminID string `json:"admin_id"`
	Version int64  `json:"version"`
	Expires int64  `json:"expires"`
}

func (a *API) signSession(claims sessionClaims) string {
	data, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(data)
	mac := hmac.New(sha256.New, []byte(a.config.SessionPassword))
	mac.Write([]byte(payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func (a *API) sessionClaims(r *http.Request) (sessionClaims, bool) {
	var claims sessionClaims
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || len(cookie.Value) > 2048 {
		return claims, false
	}
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 2 {
		return claims, false
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return claims, false
	}
	mac := hmac.New(sha256.New, []byte(a.config.SessionPassword))
	mac.Write([]byte(parts[0]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return claims, false
	}
	data, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return claims, false
	}
	if json.Unmarshal(data, &claims) != nil || claims.ID == "" || claims.AdminID == "" || claims.Version < 1 || claims.Expires <= time.Now().Unix() {
		return claims, false
	}
	return claims, true
}
func (a *API) setSession(w http.ResponseWriter, admin adminRecord) {
	expires := time.Now().Add(sessionLifetime)
	claims := sessionClaims{ID: database.NewID(), AdminID: admin.ID, Version: admin.SessionVersion, Expires: expires.Unix()}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: a.signSession(claims), Path: "/", HttpOnly: true, Secure: a.config.Production, SameSite: http.SameSiteLaxMode, Expires: expires, MaxAge: int(sessionLifetime.Seconds())})
}
func (a *API) clearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", HttpOnly: true, Secure: a.config.Production, SameSite: http.SameSiteLaxMode, Expires: time.Unix(1, 0), MaxAge: -1})
}
func (a *API) authenticated(r *http.Request) (adminRecord, sessionClaims, error) {
	claims, ok := a.sessionClaims(r)
	if !ok {
		return adminRecord{}, claims, problem.New(401, "请先登录")
	}
	admin, err := a.readAdmin()
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return admin, claims, problem.New(500, "无法读取管理员信息")
	}
	if err != nil || claims.AdminID != admin.ID || claims.Version != admin.SessionVersion {
		return admin, claims, problem.New(401, "会话已失效，请重新登录")
	}
	return admin, claims, nil
}
func (a *API) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin, _, err := a.authenticated(r)
		if err != nil {
			a.clearSession(w)
			writeError(w, err)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), adminContextKey{}, admin)))
	}
}
func (a *API) getSession(w http.ResponseWriter, r *http.Request) {
	admin, claims, err := a.authenticated(r)
	if err != nil {
		var api *problem.Error
		if errors.As(err, &api) && api.Status == 401 {
			a.clearSession(w)
			writeJSON(w, 200, map[string]any{})
			return
		}
		writeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"id": claims.ID, "user": admin.publicUser})
}
func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	admin, _, err := a.authenticated(r)
	if err != nil {
		var api *problem.Error
		if !errors.As(err, &api) || api.Status != 401 {
			writeError(w, err)
			return
		}
	} else {
		// Revoke the observed version only, so a repeated logout cannot revoke a newer login.
		if _, err := a.db.Exec(`UPDATE users SET session_version=session_version+1,updated_at=? WHERE id=? AND session_version=?`, time.Now().UnixMilli(), admin.ID, admin.SessionVersion); err != nil {
			writeError(w, problem.New(500, "无法退出登录，请重试"))
			return
		}
	}
	a.clearSession(w)
	writeJSON(w, 200, map[string]any{"loggedOut": true})
}

var usernamePattern = regexp.MustCompile(`^[a-z]{4,32}$`)

func validateUsername(value *string) error {
	if value == nil {
		return problem.Validation("username", "请输入用户名")
	}
	*value = strings.TrimSpace(*value)
	if !usernamePattern.MatchString(*value) {
		return problem.Validation("username", "用户名须为 4–32 个小写英文字母")
	}
	return nil
}
func validatePassword(value *string, field string) error {
	if value == nil || textutil.Length(*value) < 6 || textutil.Length(*value) > 32 {
		return problem.Validation(field, "密码须为 6–32 个字符")
	}
	return nil
}
func parseProfileField(raw json.RawMessage, field string, required bool) (*string, error) {
	if len(raw) == 0 {
		if required {
			return nil, problem.Validation(field, "请提供此字段")
		}
		return nil, nil
	}
	if string(raw) == "null" {
		return nil, nil
	}
	var value string
	if json.Unmarshal(raw, &value) != nil {
		return nil, problem.Validation(field, "请输入有效文本")
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	if strings.ContainsRune(value, 0) {
		return nil, problem.Validation(field, "不能包含空字符")
	}
	if field == "nickname" {
		if textutil.Length(value) > 32 {
			return nil, problem.Validation(field, "昵称最多 32 个字符")
		}
	} else {
		if textutil.Length(value) > 2048 {
			return nil, problem.Validation(field, "头像地址最多 2048 个字符")
		}
		parsed, err := url.Parse(value)
		if err != nil || !strings.EqualFold(parsed.Scheme, "https") || parsed.Hostname() == "" || parsed.User != nil {
			return nil, problem.Validation(field, "请输入不含用户名和密码的 HTTPS 地址")
		}
	}
	return &value, nil
}
func (a *API) setupStatus(w http.ResponseWriter, r *http.Request) {
	_, err := a.readAdmin()
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeError(w, problem.New(500, "无法读取管理员信息"))
		return
	}
	writeJSON(w, 200, map[string]any{"data": map[string]any{"needs_setup": errors.Is(err, sql.ErrNoRows)}})
}
func (a *API) setup(w http.ResponseWriter, r *http.Request) {
	_, err := a.readAdmin()
	if err == nil {
		writeError(w, problem.New(409, "管理员已完成初始化"))
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		writeError(w, problem.New(500, "无法读取管理员信息"))
		return
	}
	var input struct {
		Username  *string         `json:"username"`
		Password  *string         `json:"password"`
		Confirm   *string         `json:"password_confirm"`
		Nickname  json.RawMessage `json:"nickname"`
		AvatarURL json.RawMessage `json:"avatar_url"`
	}
	if err := decodeBody(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	if err := validateUsername(input.Username); err != nil {
		writeError(w, err)
		return
	}
	if err := validatePassword(input.Password, "password"); err != nil {
		writeError(w, err)
		return
	}
	if input.Confirm == nil || *input.Confirm != *input.Password {
		writeError(w, problem.Validation("password_confirm", "两次密码不一致"))
		return
	}
	nickname, err := parseProfileField(input.Nickname, "nickname", false)
	if err != nil {
		writeError(w, err)
		return
	}
	avatar, err := parseProfileField(input.AvatarURL, "avatar_url", false)
	if err != nil {
		writeError(w, err)
		return
	}
	hash, err := password.Hash(*input.Password)
	if err != nil {
		writeError(w, problem.New(500, "无法创建管理员"))
		return
	}
	admin := adminRecord{publicUser: publicUser{ID: database.NewID(), Username: *input.Username, Nickname: nickname, AvatarURL: avatar}, SessionVersion: 1}
	now := time.Now().UnixMilli()
	result, err := a.db.Exec(`INSERT INTO users(id,username,password_hash,nickname,avatar_url,created_at,updated_at) VALUES(?,?,?,?,?,?,?) ON CONFLICT DO NOTHING`, admin.ID, admin.Username, hash, nickname, avatar, now, now)
	if err != nil {
		writeError(w, problem.New(500, "无法创建管理员"))
		return
	}
	count, err := result.RowsAffected()
	if err != nil {
		writeError(w, problem.New(500, "无法创建管理员"))
		return
	}
	if count != 1 {
		writeError(w, problem.New(409, "管理员已完成初始化"))
		return
	}
	a.setSession(w, admin)
	writeJSON(w, 201, map[string]any{"data": admin.publicUser})
}
func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username *string `json:"username"`
		Password *string `json:"password"`
	}
	if err := decodeBody(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	if err := validateUsername(input.Username); err != nil {
		writeError(w, err)
		return
	}
	if err := validatePassword(input.Password, "password"); err != nil {
		writeError(w, err)
		return
	}
	admin, err := a.readAdmin()
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeError(w, problem.New(500, "无法读取管理员信息"))
		return
	}
	if err != nil || !password.Verify(admin.PasswordHash, *input.Password) || admin.Username != *input.Username {
		writeError(w, problem.New(401, "用户名或密码错误"))
		return
	}
	a.setSession(w, admin)
	writeJSON(w, 200, map[string]any{"data": admin.publicUser})
}
func (a *API) updateProfile(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Nickname  json.RawMessage `json:"nickname"`
		AvatarURL json.RawMessage `json:"avatar_url"`
	}
	if err := decodeBody(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	nickname, err := parseProfileField(input.Nickname, "nickname", true)
	if err != nil {
		writeError(w, err)
		return
	}
	avatar, err := parseProfileField(input.AvatarURL, "avatar_url", true)
	if err != nil {
		writeError(w, err)
		return
	}
	admin := currentAdmin(r)
	result, err := a.db.Exec(`UPDATE users SET nickname=?,avatar_url=?,updated_at=? WHERE id=? AND session_version=?`, nickname, avatar, time.Now().UnixMilli(), admin.ID, admin.SessionVersion)
	if err != nil {
		writeError(w, problem.New(500, "资料保存失败"))
		return
	}
	count, err := result.RowsAffected()
	if err != nil {
		writeError(w, problem.New(500, "资料保存失败"))
		return
	}
	if count != 1 {
		a.clearSession(w)
		writeError(w, problem.New(401, "会话已失效，请重新登录"))
		return
	}
	admin.Nickname = nickname
	admin.AvatarURL = avatar
	writeJSON(w, 200, map[string]any{"data": admin.publicUser})
}
func (a *API) updatePassword(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Current *string `json:"current_password"`
		New     *string `json:"new_password"`
		Confirm *string `json:"new_password_confirm"`
	}
	if err := decodeBody(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	if err := validatePassword(input.Current, "current_password"); err != nil {
		writeError(w, err)
		return
	}
	if err := validatePassword(input.New, "new_password"); err != nil {
		writeError(w, err)
		return
	}
	if input.Confirm == nil || *input.Confirm != *input.New {
		writeError(w, problem.Validation("new_password_confirm", "两次密码不一致"))
		return
	}
	admin := currentAdmin(r)
	if !password.Verify(admin.PasswordHash, *input.Current) {
		writeError(w, problem.Validation("current_password", "当前密码错误"))
		return
	}
	hash, err := password.Hash(*input.New)
	if err != nil {
		writeError(w, problem.New(500, "密码修改失败"))
		return
	}
	result, err := a.db.Exec(`UPDATE users SET password_hash=?,session_version=session_version+1,updated_at=? WHERE id=? AND session_version=?`, hash, time.Now().UnixMilli(), admin.ID, admin.SessionVersion)
	if err != nil {
		writeError(w, problem.New(500, "密码修改失败"))
		return
	}
	count, err := result.RowsAffected()
	if err != nil {
		writeError(w, problem.New(500, "密码修改失败"))
		return
	}
	if count != 1 {
		a.clearSession(w)
		writeError(w, problem.New(401, "会话已失效，请重新登录"))
		return
	}
	a.clearSession(w)
	w.WriteHeader(204)
}
