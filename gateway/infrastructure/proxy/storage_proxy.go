package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

// StorageProxy forwards Gateway requests to the internal Storage service.
type StorageProxy struct {
	proxy *httputil.ReverseProxy
}

// NewStorageProxy creates a reverse proxy targeting Storage.
//
// The service-authentication headers are always overwritten here so an
// external client cannot supply or control trusted Gateway-to-Storage
// credentials.
func NewStorageProxy(
	storageServiceURL string,
	storageServiceName string,
	storageServiceSecret string,
) (*StorageProxy, error) {
	target, err := url.Parse(storageServiceURL)
	if err != nil {
		return nil, err
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(target)

	originalDirector := reverseProxy.Director

	reverseProxy.Director = func(r *http.Request) {
		originalDirector(r)

		r.Header.Set(serviceNameHeader, storageServiceName)
		r.Header.Set(serviceSecretHeader, storageServiceSecret)
	}

	return &StorageProxy{
		proxy: reverseProxy,
	}, nil
}

// ServeHTTP forwards the incoming request.
func (p *StorageProxy) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	p.proxy.ServeHTTP(w, r)
}
