package handler

import (
	"bytes"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AI responses are bounded JSON. Hold them until application outcome accounting
// commits, so a failed commit cannot be advertised as a successful paid result.
type aiResponse struct {
	gin.ResponseWriter
	body   bytes.Buffer
	status int
	size   int
}

func newAIResponse(w gin.ResponseWriter) *aiResponse {
	return &aiResponse{ResponseWriter: w, status: http.StatusOK, size: -1}
}
func (w *aiResponse) WriteHeader(status int) {
	if w.size < 0 {
		w.status = status
	}
}
func (w *aiResponse) WriteHeaderNow() {
	if w.size < 0 {
		w.size = 0
	}
}
func (w *aiResponse) Write(b []byte) (int, error) {
	w.WriteHeaderNow()
	n, err := w.body.Write(b)
	w.size += n
	return n, err
}
func (w *aiResponse) WriteString(s string) (int, error) { return w.Write([]byte(s)) }
func (w *aiResponse) Status() int                       { return w.status }
func (w *aiResponse) Size() int                         { return w.size }
func (w *aiResponse) Written() bool                     { return w.size >= 0 }
func (w *aiResponse) Flush()                            { w.WriteHeaderNow() }
func (w *aiResponse) commit() {
	w.ResponseWriter.WriteHeader(w.status)
	_, _ = w.ResponseWriter.Write(w.body.Bytes())
}
