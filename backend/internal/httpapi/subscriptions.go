package httpapi

import (
	"encoding/json"
	"net/http"

	"clyra/backend/internal/problem"
)

func subID(r *http.Request) (string, error) {
	id := r.PathValue("id")
	if !validID(id) {
		return "", problem.Validation("id", "请输入有效 ID")
	}
	return id, nil
}
func (a *API) createSubscription(w http.ResponseWriter, r *http.Request) {
	var raw map[string]json.RawMessage
	if err := decodeBody(w, r, &raw); err != nil {
		writeError(w, err)
		return
	}
	s, err := a.subscriptions.Create(raw)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 201, map[string]any{"data": s})
}
func (a *API) patchSubscription(w http.ResponseWriter, r *http.Request) {
	a.mutateSubscription(w, r, false)
}
func (a *API) renewSubscription(w http.ResponseWriter, r *http.Request) {
	a.mutateSubscription(w, r, true)
}
func (a *API) mutateSubscription(w http.ResponseWriter, r *http.Request, renew bool) {
	id, err := subID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	var raw map[string]json.RawMessage
	if err = decodeBody(w, r, &raw); err != nil {
		writeError(w, err)
		return
	}
	s, err := a.subscriptions.Mutate(id, raw, renew)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"data": s})
}
func (a *API) deleteSubscription(w http.ResponseWriter, r *http.Request) {
	id, err := subID(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err = a.subscriptions.Delete(id, r.URL.Query()); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (a *API) listSubscriptions(w http.ResponseWriter, r *http.Request) {
	result, err := a.subscriptions.List(r.URL.Query())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, result)
}
