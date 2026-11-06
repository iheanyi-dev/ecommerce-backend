package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// PaymentProxy forwards only the Payment routes explicitly exposed
// by the Gateway router.
type PaymentProxy struct {
	proxy *httputil.ReverseProxy
}

// NewPaymentProxy creates a reverse proxy targeting the Payment service.
func NewPaymentProxy(serviceURL string) (*PaymentProxy, error) {
	target, err := url.Parse(serviceURL)
	if err != nil {
		return nil, err
	}

	return &PaymentProxy{
		proxy: httputil.NewSingleHostReverseProxy(target),
	}, nil
}

// ServeHTTP forwards the incoming request.
func (p *PaymentProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.proxy.ServeHTTP(w, r)
}
