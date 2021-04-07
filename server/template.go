package server

import (
	"html/template"
	"io/fs"
	"strings"
)

type Templates map[string]*template.Template

func (t Templates) Get(name string) (tpl *template.Template, ok bool) {
	tpl, ok = t[name]

	return
}

func ParseTemplates(templatesFS fs.FS) Templates {
	templates := Templates{}

	entries, err := fs.ReadDir(templatesFS, ".")
	if err != nil {
		panic(err)
	}

	for _, entry := range entries {
		fname := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(fname, ".html") {
			continue
		}

		t := template.New(fname)
		templates[fname] = template.Must(t.ParseFS(templatesFS, "_header.html", fname))
	}

	return templates
}
