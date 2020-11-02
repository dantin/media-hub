package asset

import (
	"fmt"
	"net/http"
	"path"
	"runtime/pprof"
	"strings"

	"github.com/dantin/logger"
)

var pprofHTTPRoot string

// Expose debug profiling at the given URL path.
func servePprof(mux *http.ServeMux, serveAt string) {
	if serveAt == "" || serveAt == "-" {
		return
	}

	pprofHTTPRoot = path.Clean("/"+serveAt) + "/"
	mux.HandleFunc(pprofHTTPRoot, profileHandler)

	logger.Infof("pprof: profiling info expose at '%s'", pprofHTTPRoot)
}

func profileHandler(wrt http.ResponseWriter, req *http.Request) {
	wrt.Header().Set("X-Content-Type-Options", "nosniff")
	wrt.Header().Set("Content-Type", "text/plain; charset=utf-8")

	profileName := strings.TrimPrefix(req.URL.Path, pprofHTTPRoot)

	profile := pprof.Lookup(profileName)
	if profile == nil {
		servePprofError(wrt, http.StatusNotFound, "Unknown profile '"+profileName+"'")
		return
	}

	// Respond with the requested profile.
	profile.WriteTo(wrt, 2)
}

func servePprofError(wrt http.ResponseWriter, status int, txt string) {
	wrt.Header().Set("Content-Type", "text/plain; charset=utf-8")
	wrt.Header().Set("X-Go-Pprof", "1")
	wrt.Header().Del("Content-Disposition")
	wrt.WriteHeader(status)
	fmt.Fprintln(wrt, txt)
}
