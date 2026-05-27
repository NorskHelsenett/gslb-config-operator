package intercept

import "net/http"

type AuthInterceptor struct {
	transport  http.RoundTripper
	token      string
	authHeader string
}

func NewAuthInterceptor(base http.RoundTripper, authHeader, token string) http.RoundTripper {
	return &AuthInterceptor{
		transport:  base,
		token:      token,
		authHeader: authHeader,
	}
}

func (a *AuthInterceptor) RoundTrip(req *http.Request) (*http.Response, error) {
	/*
	* docs on roundtripper specifically mentions:
	* "RoundTrip should not modify the request"
	* therefore a clone of the request is made instead, which we mutate
	* in-order to inject the auth headers to the request safely
	 */
	newReq := req.Clone(req.Context())

	trip := a.transport
	if trip == nil {
		trip = http.DefaultTransport
	}

	if newReq.Header.Get(a.authHeader) == "" {
		newReq.Header.Set(a.authHeader, a.token)
	}

	resp, err := trip.RoundTrip(newReq)
	if err != nil {
		return resp, err
	}

	/*
		TODO: should there be a re-auth function?
			if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
				resp, err = trip.RoundTrip(req) // retry original request after re-authentication
			}
	*/

	return resp, err
}
