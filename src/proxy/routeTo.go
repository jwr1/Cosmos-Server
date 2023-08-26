package proxy

import (
	"net/http"
	"net/http/httputil" 
	"net/url"
	"strings"
	"crypto/tls"
	spa "github.com/roberthodgen/spa-server"
	"github.com/azukaar/cosmos-server/src/utils"
)


func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

func joinURLPath(a, b *url.URL) (path, rawpath string) {
	if a.RawPath == "" && b.RawPath == "" {
		return singleJoiningSlash(a.Path, b.Path), ""
	}
	// Same as singleJoiningSlash, but uses EscapedPath to determine
	// whether a slash should be added
	apath := a.EscapedPath()
	bpath := b.EscapedPath()

	aslash := strings.HasSuffix(apath, "/")
	bslash := strings.HasPrefix(bpath, "/")

	switch {
	case aslash && bslash:
		return a.Path + b.Path[1:], apath + bpath[1:]
	case !aslash && !bslash:
		return a.Path + "/" + b.Path, apath + "/" + bpath
	}
	return a.Path + b.Path, apath + bpath
}


// NewProxy takes target host and creates a reverse proxy
func NewProxy(targetHost string, AcceptInsecureHTTPSTarget bool, VerboseForwardHeader bool, DisableHeaderHardening bool, CORSOrigin string) (*httputil.ReverseProxy, error) {
	url, err := url.Parse(targetHost)
	if err != nil {
			return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(url)
	
	proxy.Director = func(req *http.Request) {
		originalScheme := "http"
		if utils.IsHTTPS {
			originalScheme = "https"
		}
		
		urlQuery := url.RawQuery
		req.URL.Scheme = url.Scheme
		req.URL.Host = url.Host
		req.URL.Path, req.URL.RawPath = joinURLPath(url, req.URL)
		if urlQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = urlQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = urlQuery + "&" + req.URL.RawQuery
		}
		
		req.Header.Set("X-Forwarded-Proto", originalScheme)
		
		if(originalScheme == "https") {
			req.Header.Set("X-Forwarded-Ssl", "on")
		}

		if CORSOrigin != "" {
			req.Header.Set("X-Forwarded-Host", url.Host)
		}

		if VerboseForwardHeader {
			req.Header.Set("X-Origin-Host", url.Host)
			req.Header.Set("Host", url.Host)
			req.Header.Set("X-Forwarded-For", utils.GetClientIP(req))
			req.Header.Set("X-Real-IP", utils.GetClientIP(req))
		}
	}

	if AcceptInsecureHTTPSTarget && url.Scheme == "https" {
		proxy.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		utils.Debug("Response from backend: " + resp.Status)
		utils.Debug("URL was " + resp.Request.URL.String())

		if CORSOrigin != "" {
			resp.Header.Del("Access-Control-Allow-Origin")
			resp.Header.Del("Access-Control-Allow-Methods")
			resp.Header.Del("Access-Control-Allow-Headers")
			resp.Header.Del("Access-Control-Allow-Credentials")
		}
		
		if !DisableHeaderHardening {
			resp.Header.Del("Strict-Transport-Security")
			resp.Header.Del("X-Content-Type-Options")
			resp.Header.Del("Content-Security-Policy")
			resp.Header.Del("X-XSS-Protection")
		}

		return nil
	}

	return proxy, nil
}


func RouteTo(route utils.ProxyRouteConfig) http.Handler {
	// initialize a reverse proxy and pass the actual backend server url here

	destination := route.Target
	routeType := route.Mode

	if(routeType == "SERVAPP" || routeType == "PROXY") {
		proxy, err := NewProxy(destination, route.AcceptInsecureHTTPSTarget, route.VerboseForwardHeader, route.DisableHeaderHardening, route.CORSOrigin)
		if err != nil {
				utils.Error("Create Route", err)
		}

		// create a handler function which uses the reverse proxy
		return proxy
	}  else if (routeType == "STATIC") {
		return http.FileServer(http.Dir(destination))
	}  else if (routeType == "SPA") {
		return spa.SpaHandler(destination, "index.html")	
	} else if(routeType == "REDIRECT") {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, destination, 302)
		})
	} else {
		utils.Error("Invalid route type", nil)
		return nil
	}
}