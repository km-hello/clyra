package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"unicode/utf8"

	"clyra/backend/internal/problem"
)

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if status != http.StatusNoContent {
		_ = json.NewEncoder(w).Encode(value)
	}
}
func writeError(w http.ResponseWriter, err error) {
	var e *problem.Error
	if !errors.As(err, &e) {
		e = &problem.Error{Status: 500, Message: "服务器暂时无法处理请求"}
	}
	body := map[string]any{"statusCode": e.Status, "message": e.Message}
	if e.Data != nil {
		body["data"] = e.Data
	}
	writeJSON(w, e.Status, body)
}
func decodeBody(w http.ResponseWriter, r *http.Request, value any) error {
	media, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" || len(params) > 1 || (len(params) == 1 && !strings.EqualFold(params["charset"], "utf-8")) {
		return problem.New(400, "请求体必须使用 application/json")
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 32*1024))
	if err != nil {
		return problem.New(400, "请求体不能超过 32 KiB")
	}
	if !utf8.Valid(body) || len(bytes.TrimSpace(body)) == 0 || bytes.TrimSpace(body)[0] != '{' {
		return problem.New(400, "请求体必须是有效 JSON 对象")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	// encoding/json otherwise accepts differently cased names for struct fields.
	if target := reflect.TypeOf(value).Elem(); target.Kind() == reflect.Struct {
		var keys map[string]json.RawMessage
		if json.Unmarshal(body, &keys) != nil {
			return problem.New(400, "请求体必须是有效 JSON")
		}
		allowed := make(map[string]bool, target.NumField())
		for i := 0; i < target.NumField(); i++ {
			allowed[strings.Split(target.Field(i).Tag.Get("json"), ",")[0]] = true
		}
		for key := range keys {
			if !allowed[key] {
				return problem.Validation(key, "不支持的字段")
			}
		}
	}
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(value); err != nil {
		return problem.Validation("_form", "请求体必须是有效 JSON，且不能包含未知字段")
	}
	if decoder.Decode(new(any)) != io.EOF {
		return problem.New(400, "请求体必须是有效 JSON")
	}
	return nil
}

var uuidPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func validID(id string) bool { return uuidPattern.MatchString(id) }
