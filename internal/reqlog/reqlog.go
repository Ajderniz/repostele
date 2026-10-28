// Package reqlog ties error logs to the request that caused them: each
// request gets a number and, once known, a user, both shown on the request
// line and on every error logged while serving it.
package reqlog

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync/atomic"

	"github.com/go-chi/httplog/v3"
)

type ctxKey struct{}

type info struct {
  id   uint64
  user string
}

var nextID atomic.Uint64

// Middleware must run inside httplog.RequestLogger so the request line gets
// the attributes set here.
func Middleware(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    in := &info{id: nextID.Add(1)}
    ctx := context.WithValue(r.Context(), ctxKey{}, in)
    httplog.SetAttrs(ctx, slog.Uint64("req", in.id))
    next.ServeHTTP(w, r.WithContext(ctx))
  })
}

// SetUser records who made the request; only the first call counts.
func SetUser(ctx context.Context, user string) {
  in, ok := ctx.Value(ctxKey{}).(*info)
  if !ok || in.user != "" || user == "" { return }
  in.user = user
  httplog.SetAttrs(ctx, slog.String("user", user))
}

// Error logs err with the request's method, path, number and user. An error
// with a Detail method (models' DB errors) logs that instead of its message,
// which is meant for users.
func Error(r *http.Request, err error) {
  if err == nil { return }
  msg := err.Error()
  var d interface{ Detail() string }
  if errors.As(err, &d) { msg = d.Detail() }

  args := []any{"method", r.Method, "path", r.URL.Path}
  if in, ok := r.Context().Value(ctxKey{}).(*info); ok {
    args = append(args, "req", in.id)
    if in.user != "" { args = append(args, "user", in.user) }
  }
  slog.ErrorContext(r.Context(), msg, args...)
}
