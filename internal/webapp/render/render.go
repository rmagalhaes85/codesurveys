package render

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
)

type Renderer struct {
	fs fs.FS
	baseDir string
	cache map[string]*template.Template
}

type M map[string]any

func NewRenderer(efs embed.FS, baseDir string) (*Renderer, error) {
	sub, err := fs.Sub(efs, baseDir)
	if err != nil {
		return nil, fmt.Errorf("templates subfs: %w", err)
	}
	r := &Renderer{
		fs: sub,
		baseDir: baseDir,
		cache: make(map[string]*template.Template),
	}
	if err := r.buildCache(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Renderer) buildCache() error {
	pages, err := fs.Glob(r.fs, "*.tmpl")
	if err != nil {
		return err
	}
	for _, page := range pages {
		t := template.New(filepath.Base(page)) // .Funcs(r.funcs)
		t, err = t.ParseFS(r.fs,
			"base.tmpl",
			"partials/*.tmpl",
			page,
		)
		if err != nil {
			return fmt.Errorf("parse %s: %w", page, err)
		}
		r.cache[filepath.Base(page)] = t
	}
	return nil
}

func (r *Renderer) Render(w http.ResponseWriter, rqt *http.Request, name string, data M) {
	t, ok := r.cache[name]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func SubFS(efs embed.FS, path string) (fs.FS, error) {
	return fs.Sub(efs, path)
}
