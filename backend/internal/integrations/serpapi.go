package integrations

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"clyra/backend/internal/textutil"
)

type ImageSearchItem struct {
	Title     string `json:"title"`
	Original  string `json:"original"`
	Thumbnail string `json:"thumbnail"`
	Source    string `json:"source"`
}

func jsonString(value json.RawMessage, target *string) bool {
	return len(value) > 0 && string(value) != "null" && json.Unmarshal(value, target) == nil
}
func imageHTTPS(value string) bool {
	if textutil.Length(value) > 2048 {
		return false
	}
	u, e := url.Parse(value)
	return e == nil && u.Scheme == "https" && u.Host != "" && u.User == nil && !strings.ContainsAny(value, "\r\n\t")
}

type SearchError struct {
	Status  int
	Message string
}

func (e *SearchError) Error() string { return e.Message }

func SearchIcons(ctx context.Context, base *http.Client, key, query string) ([]ImageSearchItem, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, &SearchError{Status: 503, Message: "图标搜索未配置"}
	}
	u := url.URL{Scheme: "https", Host: "serpapi.com", Path: "/search.json"}
	u.RawQuery = url.Values{"engine": {"google_images"}, "q": {query}, "api_key": {key}, "safe": {"active"}, "hl": {"zh-cn"}, "ijn": {"0"}}.Encode()
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, &SearchError{Status: 502, Message: "图标搜索暂不可用，请稍后重试"}
	}
	response, err := externalClient(base).Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, &SearchError{Status: 504, Message: "图标搜索超时，请重试"}
		} else {
			return nil, &SearchError{Status: 502, Message: "图标搜索暂不可用，请稍后重试"}
		}
	}
	defer response.Body.Close()
	if response.StatusCode == 401 || response.StatusCode == 403 {
		return nil, &SearchError{Status: 503, Message: "图标搜索密钥不可用，请检查配置"}
	}
	if response.StatusCode == 429 {
		return nil, &SearchError{Status: 429, Message: "图标搜索暂不可用，请检查额度或稍后重试"}
	}
	failed := &SearchError{Status: 502, Message: "图标搜索暂不可用，请稍后重试"}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, failed
	}
	var payload map[string]json.RawMessage
	raw, readErr := io.ReadAll(io.LimitReader(response.Body, 8*1024*1024))
	if readErr != nil || json.Unmarshal(raw, &payload) != nil || payload == nil {
		return nil, failed
	}
	status := ""
	metadata, metadataPresent := payload["search_metadata"]
	if metadataPresent {
		var m map[string]json.RawMessage
		if json.Unmarshal(metadata, &m) != nil || m == nil || json.Unmarshal(m["status"], &status) != nil {
			return nil, failed
		}
		if status != "Success" {
			return nil, failed
		}
	}
	var candidates []json.RawMessage
	images, imagesPresent := payload["images_results"]
	if imagesPresent && (string(images) == "null" || json.Unmarshal(images, &candidates) != nil) {
		return nil, failed
	}
	_, errorPresent := payload["error"]
	if (errorPresent || !imagesPresent) && status != "Success" {
		return nil, failed
	}
	data := []ImageSearchItem{}
	seen := map[string]bool{}
	for _, candidate := range candidates {
		var fields map[string]json.RawMessage
		if json.Unmarshal(candidate, &fields) != nil || fields == nil {
			continue
		}
		var item ImageSearchItem
		if !jsonString(fields["title"], &item.Title) || !jsonString(fields["source"], &item.Source) || !jsonString(fields["original"], &item.Original) || !jsonString(fields["thumbnail"], &item.Thumbnail) || !imageHTTPS(item.Original) || !imageHTTPS(item.Thumbnail) || seen[item.Original] {
			continue
		}
		item.Title = textutil.Truncate(item.Title, 200)
		item.Source = textutil.Truncate(item.Source, 200)
		data = append(data, item)
		seen[item.Original] = true
		if len(data) == 100 {
			break
		}
	}
	return data, nil
}
