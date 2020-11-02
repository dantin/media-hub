package asset

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/dantin/logger"
)

func index(wrt http.ResponseWriter, req *http.Request) {
	now := time.Now().UTC().Round(time.Millisecond)
	if req.Method != http.MethodGet {
		wrt.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(wrt).Encode(ErrOperationNotAllowed(now))
		logger.Warnf("index: Invalid HTTP method %s", req.Method)
		return
	}

	wrt.Header().Set("X-Content-Type-Options", "nosniff")
	wrt.Header().Set("Content-Type", "text/json; charset=utf-8")
	io.WriteString(wrt, `{"message": "hello"}`)
}
