package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// RecommendationProxy forwards Gateway requests to the Recommendation
// service.
type RecommendationProxy struct {
	proxy *httputil.ReverseProxy
}

// NewRecommendationProxy creates a reverse proxy targeting the configured
// Recommendation service and authenticating the Gateway to Recommendation.
//
// The service-authentication headers are always overwritten here. External
// clients must never be allowed to provide trusted Gateway-to-service
// credentials.
func NewRecommendationProxy(
	recommendationServiceURL string,
	recommendationServiceName string,
	recommendationServiceSecret string,
) (*RecommendationProxy, error) {
	target, err := url.Parse(recommendationServiceURL)
	if err != nil {
		return nil, err
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(target)

	originalDirector := reverseProxy.Director

	reverseProxy.Director = func(r *http.Request) {
		originalDirector(r)

		r.Header.Set(serviceNameHeader, recommendationServiceName)
		r.Header.Set(serviceSecretHeader, recommendationServiceSecret)
	}

	return &RecommendationProxy{
		proxy: reverseProxy,
	}, nil
}

// ServeHTTP forwards the incoming request.
func (p *RecommendationProxy) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	p.proxy.ServeHTTP(w, r)
}
