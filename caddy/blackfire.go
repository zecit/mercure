// Package blackfire provides a profiler module for the Caddy Server using Blackfire.
package caddy

import (
	"net/http"
	"strings"

	"github.com/blackfireio/go-blackfire"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(Blackfire{})
	httpcaddyfile.RegisterHandlerDirective("blackfire", parseBlackfireCaddyfile)
}

// Blackfire
type Blackfire struct {
	logger *zap.Logger
}

// CaddyModule returns the Caddy module information.
func (Blackfire) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.blackfire",
		New: func() caddy.Module { return new(Blackfire) },
	}
}

func (b *Blackfire) Provision(ctx caddy.Context) error {
	b.logger = ctx.Logger(b)

	return nil
}

func (b Blackfire) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	if !strings.HasPrefix(r.URL.Path, "/_blackfire") {
		return next.ServeHTTP(w, r)
	}

	mux, err := blackfire.NewServeMux("_blackfire")
	if err != nil {
		return err
	}

	mux.ServeHTTP(w, r)

	return nil
}

// UnmarshalCaddyfile sets up the handler from Caddyfile tokens.
func (b *Blackfire) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	return nil
}

// parseBlackfireCaddyfile unmarshals tokens from h into a new Blackfire.
func parseBlackfireCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var b Blackfire
	err := b.UnmarshalCaddyfile(h.Dispenser)

	return b, err
}

var (
	_ caddyhttp.MiddlewareHandler = (*Blackfire)(nil)
	_ caddyfile.Unmarshaler       = (*Blackfire)(nil)
)
