package prpl

import (
	"bytes"
	"log"
	"strings"

	"net/http"
)

var (
	CacheImmutable    = "public, max-age=31536000, immutable"
	CacheNever        = "public, max-age=0"
	CacheNeverPrivate = "private, max-age=0"
)

func (p *PRPL) createHandler() http.Handler {
	m := http.NewServeMux()

	for _, build := range p.builds {
		m.HandleFunc(p.version+build.entrypoint, p.routeHandler)
		for path, handler := range p.staticHandlers {
			m.Handle(p.version+build.name+"/"+path, handler)
		}
	}

	m.Handle(p.version, http.StripPrefix(p.version, p.staticHandler(http.FileServer(p.root))))

	m.HandleFunc("/", p.routeHandler)

	return m
}

func (p *PRPL) routeHandler(w http.ResponseWriter, r *http.Request) {
	capabilities := p.browserCapabilities(r.UserAgent())
	build := p.builds.findBuild(capabilities)
	if build == nil {
		http.Error(w, "This browser is not supported", http.StatusInternalServerError)
		return
	}

	h := w.Header()
	h.Set("Cache-Control", CacheNever)
	build.addHeaders(p, w, r)
	build.template.Render(w, r)
}

func (p *PRPL) staticHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// TODO: Service worker location should be configurable.
		h := w.Header()
		if strings.HasSuffix(r.URL.Path, "service-worker.js") {
			h.Set("Service-Worker-Allowed", "/")
			h.Set("Cache-Control", CacheNeverPrivate)
		} else {
			h.Set("Cache-Control", CacheImmutable)
		}

		capabilities := p.browserCapabilities(r.UserAgent())
		build := p.builds.findBuild(capabilities)
		if build == nil {
			http.Error(w, "This browser is not supported", http.StatusInternalServerError)
			return
		}

		build.addHeaders(p, w, r)

		file, found := files[r.URL.Path]
		if !found {
			next.ServeHTTP(w, r)
			return
		}

		content := bytes.NewReader(file.data)
		http.ServeContent(w, r, r.URL.Path, file.modTime, content)
	}

	return http.HandlerFunc(fn)
}

func (b *build) addHeaders(p *PRPL, w http.ResponseWriter, r *http.Request) {
	header := w.Header()
	filename := r.URL.Path

	log.Printf("filename: %q", filename)
	for f := range b.pushHeaders {
		log.Printf("f: %q", f)
	}

	links, ok := b.pushHeaders[filename]
	if !ok {
		links, ok = b.pushHeaders[p.version+filename]
		if !ok {
			return
		}
	}

	if pusher, ok := w.(http.Pusher); ok && p.shouldPush != nil && p.shouldPush(r) {
		for _, link := range links {
			pusher.Push(link.file, &http.PushOptions{
				Header: http.Header{
					"Cache-Control": []string{CacheImmutable},
					//"Content-Type":  []string{"TODO: file content type here"},
				},
			})
		}
	}

	for _, link := range links {
		header.Add("Link", link.String())
	}
}
