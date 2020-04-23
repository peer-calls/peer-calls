package server

import (
	"net/http"

	"github.com/oxtoacart/bpool"
)

type Renderer struct {
	log Logger

	bufPool   *bpool.BufferPool
	templates Templates
	Version   string
	BaseURL   string
}

func NewRenderer(loggerFactory LoggerFactory, templates Templates, baseURL string, version string) *Renderer {
	return &Renderer{
		log:       loggerFactory.GetLogger("renderer"),
		bufPool:   bpool.NewBufferPool(128),
		templates: templates,
		Version:   version,
		BaseURL:   baseURL,
	}
}

type PageHandler func(
	w http.ResponseWriter,
	r *http.Request,
) (templateName string, data interface{}, err error)

func (tr *Renderer) Render(h PageHandler) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		templateName, data, err := h(w, r)
		if err == nil && templateName == "" {
			return
		}
		template, ok := tr.templates.Get(templateName)
		if !ok {
			tr.log.Println("Template not found:", templateName)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err != nil {
			tr.log.Println("An error occurred:", err)
			w.WriteHeader(http.StatusInternalServerError)
		}

		dataMap := map[string]interface{}{
			"Data":    data,
			"BaseURL": tr.BaseURL,
			"Version": tr.Version,
		}

		buf := tr.bufPool.Get()
		defer tr.bufPool.Put(buf)
		tr.log.Println("Rendering template:", templateName)
		err = template.Execute(buf, dataMap)
		if err != nil {
			tr.log.Println("Error rendering template", templateName, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		buf.WriteTo(w)
	}
	return http.HandlerFunc(fn)
}
