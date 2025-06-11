package middleware

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/ASRafalsky/telemetry/pkg/log"
	"go.uber.org/multierr"
)

type (
	responseData struct {
		status int
	}

	loggingResponseWriter struct {
		http.ResponseWriter
		responseData responseData
		err          error
	}

	loggingReadCloser struct {
		io.ReadCloser
		err error
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	if err != nil {
		r.err = err
	}
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

func (r *loggingReadCloser) Read(b []byte) (int, error) {
	n, err := r.ReadCloser.Read(b)
	if err != nil && err != io.EOF {
		r.err = err
	}
	return n, err
}

func (r *loggingReadCloser) Close() error {
	err := r.ReadCloser.Close()
	if err != nil {
		r.err = multierr.Append(r.err, err)
	}
	return r.err
}

func WithLogging(h http.Handler, l *log.Logger) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		url := r.URL.String()
		method := r.Method

		lw := loggingResponseWriter{ResponseWriter: w}
		lr := loggingReadCloser{ReadCloser: r.Body}
		r.Body = &lr
		h.ServeHTTP(&lw, r)

		duration := time.Since(start)

		l.Debug("[Handler/Request]",
			"url:", url,
			"method:", method,
			"duration:", duration.String(),
		)
		l.Debug("[Handler/Response]", "status:", strconv.Itoa(lw.responseData.status))
		if lw.err != nil {
			l.Error("[Handler/Error]", lw.err.Error())
		}
		if lr.err != nil {
			l.Error("[Handler/Error]", lr.err.Error())
		}
	}
	return http.HandlerFunc(logFn)
}
