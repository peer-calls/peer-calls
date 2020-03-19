package render

import (
	"log"
	"net/http"
	"os"

	"github.com/oxtoacart/bpool"
)

type Renderer struct {
	bufPool   *bpool.BufferPool
	logger    *log.Logger
	templates Templates
	Version   string
	BaseURL   string
}

func NewRenderer(templates Templates, baseURL string, version string) *Renderer {
	return &Renderer{
		bufPool:   bpool.NewBufferPool(128),
		logger:    log.New(os.Stdout, "REND ", log.LstdFlags),
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
			tr.logger.Println("Template not found:", templateName)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err != nil {
			tr.logger.Println("An error occurred:", err)
			w.WriteHeader(http.StatusInternalServerError)
		}

		dataMap := map[string]interface{}{
			"Data":    data,
			"BaseURL": tr.BaseURL,
			"Version": tr.Version,
		}

		buf := tr.bufPool.Get()
		defer tr.bufPool.Put(buf)
		tr.logger.Println("Rendering template:", templateName)
		err = template.Execute(buf, dataMap)
		if err != nil {
			tr.logger.Println("Error rendering template", templateName, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		buf.WriteTo(w)
	}
	return http.HandlerFunc(fn)
}
