package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLoggerUsesCloudLoggingFieldNames(t *testing.T) {
	var buf bytes.Buffer
	NewLogger(&buf).Warn("hello", "n", 1)
	var line map[string]any
	if err := json.Unmarshal(buf.Bytes(), &line); err != nil {
		t.Fatalf("log line is not JSON: %v", err)
	}
	if line["severity"] != "WARN" || line["message"] != "hello" {
		t.Errorf("log line = %v, want severity WARN and message hello", line)
	}
}

func TestMiddlewareAddsTraceAndLogsStatus(t *testing.T) {
	var buf bytes.Buffer
	h := Middleware(NewLogger(&buf), "my-project", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Logger(r.Context()).Info("inside handler")
		w.WriteHeader(http.StatusTeapot)
	}))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-Cloud-Trace-Context", "abc123/456;o=1")
	h.ServeHTTP(httptest.NewRecorder(), req)

	out := buf.String()
	if !strings.Contains(out, `"logging.googleapis.com/trace":"projects/my-project/traces/abc123"`) {
		t.Errorf("logs do not carry the trace ID:\n%s", out)
	}
	if !strings.Contains(out, `"status":418`) {
		t.Errorf("request log does not record status 418:\n%s", out)
	}
}

func TestMiddlewareRecoversFromPanic(t *testing.T) {
	h := Middleware(NewLogger(io.Discard), "", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	var body ErrorBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil || body.Error.Code != "internal" {
		t.Errorf("body = %+v (%v), want error code internal", body, err)
	}
}

func TestShutdownLetsRequestInProgressFinish(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0") // borrow a free port
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()

	started := make(chan struct{})
	srv := &http.Server{Addr: addr, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		time.Sleep(200 * time.Millisecond) // a slow request still running when shutdown begins
		io.WriteString(w, "finished")
	})}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- serveUntil(ctx, srv, NewLogger(io.Discard), 5*time.Second) }()

	respCh := make(chan string, 1)
	go func() {
		for i := 0; i < 50; i++ { // wait for the server to start listening
			resp, err := http.Get("http://" + addr)
			if err == nil {
				b, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				respCh <- string(b)
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		respCh <- "never connected"
	}()

	<-started
	cancel() // like SIGTERM arriving mid-request

	if got := <-respCh; got != "finished" {
		t.Errorf("response = %q, want %q: the in-progress request was cut off", got, "finished")
	}
	if err := <-done; err != nil {
		t.Errorf("serveUntil() error = %v, want nil after a clean shutdown", err)
	}
}
