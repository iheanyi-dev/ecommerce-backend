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

// ProductProxy forwards Gateway requests to the Product service.
type ProductProxy struct {
	proxy *httputil.ReverseProxy
}

// NewProductProxy creates a reverse proxy targeting the configured Product
// service and authenticating the Gateway to Product.
//
// The service-authentication headers are always overwritten here. This is
// important because those headers represent trusted Gateway-to-Store
// credentials and must never be accepted from an external client.
func NewProductProxy(
	productServiceURL string,
	productServiceName string,
	productServiceSecret string,
) (*ProductProxy, error) {
	target, err := url.Parse(productServiceURL)
	if err != nil {
		return nil, err
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(target)

	originalDirector := reverseProxy.Director

	reverseProxy.Director = func(r *http.Request) {
		originalDirector(r)

		r.Header.Set(serviceNameHeader, productServiceName)
		r.Header.Set(serviceSecretHeader, productServiceSecret)
	}

	return &ProductProxy{
		proxy: reverseProxy,
	}, nil
}

// ServeHTTP forwards the incoming request.
func (p *ProductProxy) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	p.proxy.ServeHTTP(w, r)
}
