package http

import (
	"net/http"
)

func getIDFromRequest(r *http.Request) (string, bool) {
	requestID, ok := r.Context().Value(requestIDContextKey).(string)
	if !ok || len(requestID) == 0 {
		return "", false
	}
	return requestID, true
}
