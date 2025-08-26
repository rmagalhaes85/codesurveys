package webapp

import (
	"context"
	"embed"
	"log"
	"net/http"
	"time"

	"github.com/rmagalhaes85/codesurveys/internal/webapp/handlers"
	"github.com/rmagalhaes85/codesurveys/internal/webapp/models"
	"github.com/rmagalhaes85/codesurveys/internal/webapp/render"
)

//go:embed templates static
var embedFS embed.FS

type Server struct {
	*http.Server
}

func New(ctx context.Context, dsn string) (*Server, error) {
	store, err := models.NewPostgresStore(ctx, dsn)
	if err != nil {
		return nil, err
	}

	tpl, err := render.NewRenderer(embedFS, "templates")
	if err != nil {
		log.Fatalf("template init error: %v", err)
	}

	staticFS, err := render.SubFS(embedFS, "static")
	if err != nil {
		log.Fatalf("static fs: %w", err)
	}
	static := http.FileServerFS(staticFS)
	_ = static

	hdlr := handlers.New(store, tpl)

	mux := http.NewServeMux()
	mux.Handle("GET /", hdlr.Home())

	root := requestLogger(recoverer(mux))

	return &Server{
		Server: &http.Server {
			Addr: ":8080",
			Handler: root,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}, nil
}

func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic: %v", rec)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		st := time.Now()
		ww := &wrap{ResponseWriter: w, status: 200}
		next.ServeHTTP(ww, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, ww.status, time.Since(st))
	})
}

type wrap struct {
	http.ResponseWriter
	status int
}

func (w *wrap) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
