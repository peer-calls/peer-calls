package render

import (
	"net/http"

	"github.com/jeremija/peer-calls/src/server-go/logger"
	"github.com/oxtoacart/bpool"
)

type Renderer struct {
	bufPool   *bpool.BufferPool
	templates Templates
	Version   string
	BaseURL   string
}

var log = logger.GetLogger("render")

func NewRenderer(templates Templates, baseURL string, version string) *Renderer {
	return &Renderer{
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
			log.Println("Template not found:", templateName)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err != nil {
			log.Println("An error occurred:", err)
			w.WriteHeader(http.StatusInternalServerError)
		}

		dataMap := map[string]interface{}{
			"Data":    data,
			"BaseURL": tr.BaseURL,
			"Version": tr.Version,
		}

		buf := tr.bufPool.Get()
		defer tr.bufPool.Put(buf)
		log.Println("Rendering template:", templateName)
		err = template.Execute(buf, dataMap)
		if err != nil {
			log.Println("Error rendering template", templateName, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		buf.WriteTo(w)
	}
	return http.HandlerFunc(fn)
}
