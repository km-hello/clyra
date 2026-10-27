package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"clyra/backend/internal/integrations"
	"clyra/backend/internal/problem"
	"clyra/backend/internal/textutil"
)

func (a *API) searchIcons(w http.ResponseWriter, r *http.Request) {
	var input map[string]json.RawMessage
	if err := decodeBody(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	var query string
	if json.Unmarshal(input["query"], &query) != nil {
		writeError(w, problem.Validation("query", "请输入搜索关键词"))
		return
	}
	query = strings.TrimSpace(query)
	if query == "" {
		writeError(w, problem.Validation("query", "请输入搜索关键词"))
		return
	}
	if textutil.Length(query) > 100 {
		writeError(w, problem.Validation("query", "搜索关键词不能超过 100 个字符"))
		return
	}

	source := "serpapi"
	if raw, present := input["source"]; present {
		if string(raw) == "null" || json.Unmarshal(raw, &source) != nil {
			writeError(w, problem.Validation("source", "请选择有效的图标搜索来源"))
			return
		}
	}
	var data []integrations.ImageSearchItem
	var err error
	switch source {
	case "serpapi":
		data, err = integrations.SearchIcons(r.Context(), a.HTTPClient, a.config.SerpAPIKey, query)
	case "selfhst", "dashboardicons":
		data, err = a.iconCatalogs.Search(r.Context(), a.HTTPClient, source, query)
	default:
		writeError(w, problem.Validation("source", "请选择有效的图标搜索来源"))
		return
	}
	if err != nil {
		if failure, ok := err.(*integrations.SearchError); ok {
			err = problem.New(failure.Status, failure.Message)
		}
		writeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"data": data})
}
