package server

import (
	"html/template"
	"strings"

	"github.com/gobuffalo/packd"
	"github.com/gobuffalo/packr"
)

type Templates map[string]*template.Template

func (t Templates) Get(name string) (tpl *template.Template, ok bool) {
	tpl, ok = t[name]
	return
}

func ParseTemplates(box packr.Box) Templates {
	templates := Templates{}

	box.Walk(func(filename string, file packd.File) error {
		if !strings.HasSuffix(filename, ".html") {
			return nil
		}
		t := template.New(filename)

		base, err := box.FindString("_header.html")
		template.Must(t, err)
		template.Must(t.Parse(base))

		body, err := box.FindString(filename)
		template.Must(t, err)
		template.Must(t.Parse(body))

		templates[filename] = t

		return nil
	})

	return templates
}
