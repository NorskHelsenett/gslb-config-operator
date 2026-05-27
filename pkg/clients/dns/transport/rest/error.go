package rest

import "net/http"

type HTTPError struct {
	err      error
	Response *http.Response
}

func (e *HTTPError) Error() string {
	return e.err.Error()
}
