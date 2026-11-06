package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// OrderProxy forwards Gateway requests to the Order service.
type OrderProxy struct {
	proxy *httputil.ReverseProxy
}

// NewOrderProxy creates a reverse proxy with trusted Gateway-to-Order
// service credentials.
func NewOrderProxy(
	serviceURL string,
	serviceName string,
	serviceSecret string,
) (*OrderProxy, error) {
	target, err := url.Parse(serviceURL)
	if err != nil {
		return nil, err
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := reverseProxy.Director

	reverseProxy.Director = func(r *http.Request) {
		originalDirector(r)
		r.Header.Set(serviceNameHeader, serviceName)
		r.Header.Set(serviceSecretHeader, serviceSecret)
	}

	return &OrderProxy{proxy: reverseProxy}, nil
}

// ServeHTTP forwards the incoming request.
func (p *OrderProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.proxy.ServeHTTP(w, r)
}
