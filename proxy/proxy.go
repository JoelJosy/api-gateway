package proxy

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/JoelJosy/api-gateway/config"
	"github.com/JoelJosy/api-gateway/router"
)

// implements http.Handler interface for ListenAndServe to work
type Proxy struct {
	router    *router.Router
	transport *http.Transport
}

func NewProxy(r *router.Router, proxyCfg config.ProxyConfig) *Proxy {
	if proxyCfg.DialTimeout == 0 {
		proxyCfg.DialTimeout = 5 * time.Second
	}
	if proxyCfg.TLSHandshakeTimeout == 0 {
		proxyCfg.TLSHandshakeTimeout = 5 * time.Second
	}
	if proxyCfg.ResponseHeaderTimeout == 0 {
		proxyCfg.ResponseHeaderTimeout = 5 * time.Second
	}

	return &Proxy{
		router: r,
		transport: &http.Transport{
			// pre connection timeouts (DNS lookup, TCP connect hang)
			DialContext: (&net.Dialer{
				Timeout: proxyCfg.DialTimeout,
			}).DialContext,
			// https timeout
			TLSHandshakeTimeout: proxyCfg.TLSHandshakeTimeout,
			// post connection timeout
			// The maximum amount of time to wait for the upstream's HTTP response headers
			ResponseHeaderTimeout: proxyCfg.ResponseHeaderTimeout,
		},
	}
}

// http handler for proxy
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// get upstream url from router
	upstream, routePath, err := p.router.Match(r.URL.Path)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// parse into url structure
	targetURL, err := url.Parse(upstream)
	if err != nil {
		http.Error(w, "Invalid upstream", http.StatusInternalServerError)
		return
	}

	// create reverse proxy and define how request should be rewritten
	proxy := &httputil.ReverseProxy{}
	proxy.Rewrite = func(pr *httputil.ProxyRequest) {
		// pr.In = original incoming request (client → gateway)
		// pr.Out = request that will be sent to upstream service
		pr.SetURL(targetURL)
		// - sets scheme (http/https)
		// - sets host
		// - preserves path + query from pr.In
		// - prepares pr.Out for forwarding
		// strip /typicode prefix

		// strip prefix for upstream
		pr.Out.URL.Path = strings.TrimPrefix(pr.In.URL.Path, routePath)
		if pr.Out.URL.Path == "" {
			pr.Out.URL.Path = "/"
		}
	}

	proxy.Transport = p.transport

	// forwards request upstream and writes response to client
	proxy.ServeHTTP(w, r)
}
