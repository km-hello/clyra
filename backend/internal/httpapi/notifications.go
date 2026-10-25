package httpapi

import "net/http"

func (a *API) listNotificationRuns(w http.ResponseWriter, r *http.Request) {
	result, err := a.notifications.List()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, result)
}
func (a *API) testNotification(w http.ResponseWriter, r *http.Request) {
	result, err := a.notifications.Test()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, result)
}
