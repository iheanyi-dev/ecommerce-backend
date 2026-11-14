package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// PromotionProxy forwards Gateway requests to the Promotion service.
type PromotionProxy struct {
	proxy *httputil.ReverseProxy
}

// NewPromotionProxy creates a reverse proxy targeting Promotion.
//
// Trusted service-authentication headers are overwritten so external
// clients cannot supply Gateway-to-Promotion credentials.
func NewPromotionProxy(
	promotionServiceURL string,
	promotionServiceName string,
	promotionServiceSecret string,
) (*PromotionProxy, error) {
	target, err := url.Parse(promotionServiceURL)
	if err != nil {
		return nil, err
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := reverseProxy.Director

	reverseProxy.Director = func(r *http.Request) {
		originalDirector(r)
		r.Header.Set(serviceNameHeader, promotionServiceName)
		r.Header.Set(serviceSecretHeader, promotionServiceSecret)
	}

	// Gateway owns the external request ID response header.
	// Do not forward Promotion's copy back to the client.
	reverseProxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Del("X-Request-ID")
		return nil
	}

	return &PromotionProxy{proxy: reverseProxy}, nil
}

// ServeHTTP forwards the incoming request to Promotion.
func (p *PromotionProxy) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	p.proxy.ServeHTTP(w, r)
}
