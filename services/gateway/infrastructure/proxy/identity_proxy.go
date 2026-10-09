package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

const (
	serviceNameHeader   = "X-Service-Name"
	serviceSecretHeader = "X-Service-Secret"
)

// IdentityProxy forwards Gateway requests to the Identity service.
type IdentityProxy struct {
	proxy *httputil.ReverseProxy
}

// NewIdentityProxy creates a reverse proxy targeting the configured Identity
// service and authenticating the Gateway to Identity.
func NewIdentityProxy(
	identityServiceURL string,
	identityServiceName string,
	identityServiceSecret string,
) (*IdentityProxy, error) {
	target, err := url.Parse(identityServiceURL)
	if err != nil {
		return nil, err
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(target)

	originalDirector := reverseProxy.Director

	reverseProxy.Director = func(r *http.Request) {
		originalDirector(r)

		// Always overwrite these headers rather than trusting values supplied
		// by an external caller.
		r.Header.Set(serviceNameHeader, identityServiceName)
		r.Header.Set(serviceSecretHeader, identityServiceSecret)
	}

	return &IdentityProxy{
		proxy: reverseProxy,
	}, nil
}

// ServeHTTP forwards the incoming request.
func (p *IdentityProxy) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	p.proxy.ServeHTTP(w, r)
}
