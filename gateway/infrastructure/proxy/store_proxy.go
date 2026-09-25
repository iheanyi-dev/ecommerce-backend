package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// const (
// 	serviceNameHeader   = "X-Service-Name"
// 	serviceSecretHeader = "X-Service-Secret"
// )

// StoreProxy forwards Gateway requests to the Store service.
type StoreProxy struct {
	proxy *httputil.ReverseProxy
}

// NewStoreProxy creates a reverse proxy targeting the configured Store
// service and authenticating the Gateway to Store.
//
// The service-authentication headers are always overwritten here. This is
// important because those headers represent trusted Gateway-to-Store
// credentials and must never be accepted from an external client.
func NewStoreProxy(
	storeServiceURL string,
	storeServiceName string,
	storeServiceSecret string,
) (*StoreProxy, error) {
	target, err := url.Parse(storeServiceURL)
	if err != nil {
		return nil, err
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(target)

	originalDirector := reverseProxy.Director

	reverseProxy.Director = func(r *http.Request) {
		originalDirector(r)

		r.Header.Set(serviceNameHeader, storeServiceName)
		r.Header.Set(serviceSecretHeader, storeServiceSecret)
	}

	return &StoreProxy{
		proxy: reverseProxy,
	}, nil
}

// ServeHTTP forwards the incoming request.
func (p *StoreProxy) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	p.proxy.ServeHTTP(w, r)
}
