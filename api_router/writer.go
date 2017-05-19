package api_router

import (
	"bytes"
	"io"
	"net/http"

	"github.com/felixge/httpsnoop"
)

type ResponseTracker interface {
	http.ResponseWriter
	SetStatus(int)
	WriteStatusHeader()
	Status() int
	Size() int64
	Response() []byte
}

type responseTracker struct {
	http.ResponseWriter
	storeResponse bool
	statusWritten bool
	defaultStatus int
	status        int
	size          int64
	response      []byte
}

func (rw *responseTracker) writeStatusHeader() {
	if rw.status == 0 {
		rw.status = rw.defaultStatus
	}
	rw.WriteHeader(rw.status)
	rw.statusWritten = true
}

func (rw *responseTracker) WriteStatusHeader() {
	if !rw.statusWritten {
		rw.writeStatusHeader()
	}
}

func (rw *responseTracker) Status() int {
	return rw.status
}

func (rw *responseTracker) Size() int64 {
	return rw.size
}

func (rw *responseTracker) SetStatus(status int) {
	if !rw.statusWritten {
		rw.status = status
	}
}

func (rw *responseTracker) Response() []byte {
	return rw.response
}

func newResponseTracker(w http.ResponseWriter, default_status int, store_response bool) *responseTracker {
	rw := &responseTracker{
		defaultStatus: default_status,
		storeResponse: store_response,
	}

	hooks := httpsnoop.Hooks{
		WriteHeader: func(next httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
			return func(code int) {
				if !rw.statusWritten {
					next(code)
					rw.status = code
					rw.statusWritten = true
				}
			}
		},

		Write: func(next httpsnoop.WriteFunc) httpsnoop.WriteFunc {
			return func(p []byte) (int, error) {
				if !rw.statusWritten {
					rw.writeStatusHeader()
				}
				n, err := next(p)
				if n >= 0 {
					if rw.storeResponse {
						rw.response = append(rw.response, p[:n]...)
					}
					rw.size += int64(n)
				}
				return n, err
			}
		},

		ReadFrom: func(next httpsnoop.ReadFromFunc) httpsnoop.ReadFromFunc {
			return func(src io.Reader) (int64, error) {
				if !rw.statusWritten {
					rw.writeStatusHeader()
				}

				if rw.storeResponse {
					var buf bytes.Buffer
					_, err := io.Copy(&buf, src)
					if err != nil {
						return 0, err
					}
					rw.response = append(rw.response, buf.Bytes()...)
					return next(&buf)
				}

				n, err := next(src)
				if n >= 0 {
					rw.size += int64(n)
				}
				return n, err
			}
		},
	}

	rw.ResponseWriter = httpsnoop.Wrap(w, hooks)
	return rw
}
