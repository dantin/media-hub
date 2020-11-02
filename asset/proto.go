package asset

import (
	"net/http"
	"time"
)

// ServerCtrlResp is a server control response {ctrl}.
type ServerCtrlResp struct {
	Code      int       `json:"code"`
	Text      string    `json:"text,omitempty"`
	Timestamp time.Time `json:"ts"`
}

// ServerResp is a wrapper for server side response.
type ServerResp struct {
	Ctrl *ServerCtrlResp `json:"ctrl,omitempty"`
}

// ErrOperationNotAllowed a valid operation is not permitted in this context (405).
func ErrOperationNotAllowed(ts time.Time) *ServerResp {
	return &ServerResp{Ctrl: &ServerCtrlResp{
		Code:      http.StatusMethodNotAllowed, // 405
		Text:      "operation or method not allowed",
		Timestamp: ts,
	}}
}
