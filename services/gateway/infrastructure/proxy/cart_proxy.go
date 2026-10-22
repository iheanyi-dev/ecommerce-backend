package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// CartProxy forwards Gateway requests to the Cart service.
type CartProxy struct {
	proxy *httputil.ReverseProxy
}

// NewCartProxy creates a reverse proxy targeting the configured Cart
// service and authenticating the Gateway to Cart.
//
// Service-authentication headers are always overwritten so external clients
// cannot supply trusted Gateway-to-Cart credentials.
func NewCartProxy(
	cartServiceURL string,
	cartServiceName string,
	cartServiceSecret string,
) (*CartProxy, error) {
	target, err := url.Parse(cartServiceURL)
	if err != nil {
		return nil, err
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(target)

	originalDirector := reverseProxy.Director

	reverseProxy.Director = func(r *http.Request) {
		originalDirector(r)

		r.Header.Set(serviceNameHeader, cartServiceName)
		r.Header.Set(serviceSecretHeader, cartServiceSecret)
	}

	return &CartProxy{
		proxy: reverseProxy,
	}, nil
}

// ServeHTTP forwards the incoming request.
func (p *CartProxy) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	p.proxy.ServeHTTP(w, r)
}
