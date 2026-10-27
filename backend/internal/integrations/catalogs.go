package integrations

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"clyra/backend/internal/textutil"
)

const catalogTTL = 24 * time.Hour
const catalogMaxBytes = 8 * 1024 * 1024

var catalogFilename = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

type catalogIcon struct {
	item       ImageSearchItem
	name, slug string
}

type catalogIndex struct {
	lock    chan struct{}
	icons   []catalogIcon
	expires time.Time
}

type CatalogSearch struct {
	indexes map[string]*catalogIndex
}

func NewCatalogSearch() *CatalogSearch {
	return &CatalogSearch{indexes: map[string]*catalogIndex{
		"selfhst":        {lock: make(chan struct{}, 1)},
		"dashboardicons": {lock: make(chan struct{}, 1)},
	}}
}

func normalizeIconName(value string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return unicode.IsSpace(r) || r == '-' || r == '_'
	}), " ")
}

func (s *CatalogSearch) Search(ctx context.Context, client *http.Client, source, query string) ([]ImageSearchItem, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	index, ok := s.indexes[source]
	if !ok {
		return nil, &SearchError{Status: 400, Message: "不支持的图标搜索来源"}
	}
	query = normalizeIconName(query)
	if query == "" {
		return []ImageSearchItem{}, nil
	}
	icons, err := index.load(ctx, client, source)
	if err != nil {
		return nil, err
	}
	// Keep the cached index immutable; each request owns its filtered results.
	matches := []catalogIcon{}
	seen := map[string]bool{}
	for _, icon := range icons {
		if (strings.Contains(icon.name, query) || strings.Contains(icon.slug, query)) && !seen[icon.item.Original] {
			matches = append(matches, icon)
			seen[icon.item.Original] = true
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		exact := func(icon catalogIcon) bool { return icon.name == query || icon.slug == query }
		return exact(matches[i]) && !exact(matches[j])
	})
	if len(matches) > 100 {
		matches = matches[:100]
	}
	data := make([]ImageSearchItem, 0, len(matches))
	for _, icon := range matches {
		data = append(data, icon.item)
	}
	return data, nil
}

func catalogFailure(ctx context.Context) error {
	if ctx.Err() == context.DeadlineExceeded {
		return &SearchError{Status: 504, Message: "图标库搜索超时，请重试"}
	}
	return &SearchError{Status: 502, Message: "图标库暂不可用，请稍后重试"}
}

func (index *catalogIndex) load(ctx context.Context, client *http.Client, source string) ([]catalogIcon, error) {
	select {
	case index.lock <- struct{}{}:
		defer func() { <-index.lock }()
	case <-ctx.Done():
		return nil, catalogFailure(ctx)
	}
	if ctx.Err() != nil {
		return nil, catalogFailure(ctx)
	}
	if time.Now().Before(index.expires) {
		return index.icons, nil
	}
	indexURL := "https://raw.githubusercontent.com/selfhst/icons/main/index.json"
	if source == "dashboardicons" {
		indexURL = "https://raw.githubusercontent.com/homarr-labs/dashboard-icons/main/tree.json"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, indexURL, nil)
	if err != nil {
		return nil, catalogFailure(ctx)
	}
	response, err := externalClient(client).Do(req)
	if err != nil {
		return nil, catalogFailure(ctx)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusTooManyRequests {
		return nil, &SearchError{Status: 429, Message: "图标库请求过于频繁，请稍后重试"}
	}
	if response.StatusCode != http.StatusOK {
		return nil, catalogFailure(ctx)
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, catalogMaxBytes+1))
	if err != nil || len(raw) > catalogMaxBytes {
		return nil, catalogFailure(ctx)
	}
	icons, err := parseCatalog(source, raw)
	if err != nil || ctx.Err() != nil {
		return nil, catalogFailure(ctx)
	}
	index.icons = icons
	index.expires = time.Now().Add(catalogTTL)
	return index.icons, nil
}

func parseCatalog(source string, raw []byte) ([]catalogIcon, error) {
	icons := []catalogIcon{}
	add := func(title, slug, filename, label, repository string) {
		if !catalogFilename.MatchString(filename) {
			return
		}
		image := "https://cdn.jsdelivr.net/gh/" + repository + "@main/png/" + filename
		if !imageHTTPS(image) {
			return
		}
		icons = append(icons, catalogIcon{
			item: ImageSearchItem{Title: textutil.Truncate(title, 200), Original: image, Thumbnail: image, Source: label},
			name: normalizeIconName(title), slug: normalizeIconName(slug),
		})
	}
	if source == "selfhst" {
		var entries []struct {
			Name, Reference, PNG string
		}
		if err := json.Unmarshal(raw, &entries); err != nil {
			return nil, err
		}
		if entries == nil {
			return nil, io.ErrUnexpectedEOF
		}
		for _, entry := range entries {
			if entry.PNG == "Yes" && entry.Reference != "" {
				add(entry.Name, entry.Reference, entry.Reference+".png", "selfh.st", "selfhst/icons")
			}
		}
	} else {
		var entries struct {
			PNG []string `json:"png"`
		}
		if err := json.Unmarshal(raw, &entries); err != nil {
			return nil, err
		}
		if entries.PNG == nil {
			return nil, io.ErrUnexpectedEOF
		}
		for _, filename := range entries.PNG {
			if !strings.HasSuffix(filename, ".png") {
				continue
			}
			slug := strings.TrimSuffix(filename, ".png")
			if strings.HasSuffix(slug, "-light") || strings.HasSuffix(slug, "-dark") {
				continue
			}
			add(slug, slug, filename, "Dashboard Icons", "homarr-labs/dashboard-icons")
		}
	}
	return icons, nil
}
