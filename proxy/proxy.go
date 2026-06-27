package proxy

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/JoelJosy/api-gateway/config"
	"github.com/JoelJosy/api-gateway/router"
)

// implements http.Handler interface for ListenAndServe to work
type Proxy struct {
	router    *router.Router
	transport *http.Transport
	breakers    map[string]*CircuitBreaker
	breakersMu  sync.RWMutex
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
		breakers: make(map[string]*CircuitBreaker),
	}
}

// Helper to safely fetch or create a circuit breaker for an upstream URL
func (p *Proxy) getBreaker(upstream string) *CircuitBreaker {
	p.breakersMu.RLock()
	cb, exists := p.breakers[upstream]
	p.breakersMu.RUnlock()

	if exists {
		return cb
	}

	p.breakersMu.Lock()
	defer p.breakersMu.Unlock()

	// Double check inside the write lock to prevent race conditions
	if cb, exists = p.breakers[upstream]; exists {
		return cb
	}

	// Create new breaker for upstream
	cb = NewCircuitBreaker(5, 20*time.Second)
	p.breakers[upstream] = cb
	return cb
}


// http handler for proxy
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// get upstream url from router
	upstream, routePath, err := p.router.Match(r.URL.Path)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// get circuit breaker for upstream
	cb := p.getBreaker(upstream)

	// check circuit breaker state, switch states if needed
	if !cb.Allow() {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Circuit-Breaker", "OPEN")
		w.WriteHeader(http.StatusServiceUnavailable)
		// Custom header letting clients know a circuit breaker caught them
		w.Write([]byte(`{"error": "Upstream service is unhealthy. Circuit breaker is open."}`))
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

	// set timeout config for stransport 
	proxy.Transport = p.transport

	// upstream sends response, inspect and update cb
	proxy.ModifyResponse = func(resp *http.Response) error {
		if resp.StatusCode < 500 {
			// succesful => reset breaker state
			cb.RecordResult(nil)
		} else if resp.StatusCode == http.StatusBadGateway || resp.StatusCode == http.StatusGatewayTimeout {
			// upstream error => update breaker state
			cb.RecordResult(fmt.Errorf("upstream returned code %d", resp.StatusCode))
		}
		return nil
	}

	// unable to reach upstream (network drop)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		// Log the failure telemetry
		slog.Error("reverse proxy network failure", "upstream", upstream, "err", err)
		
		// update breaker
		cb.RecordResult(err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway) 
		w.Write([]byte(`{"error": "Bad gateway connection to upstream microservice"}`))
	} 

	// forwards request upstream and writes response to client
	proxy.ServeHTTP(w, r)
}
