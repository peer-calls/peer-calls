package server

import (
	"net/http"

	"github.com/juju/errors"
	"github.com/oxtoacart/bpool"
	"github.com/peer-calls/peer-calls/v4/server/logger"
)

const defaultBufferPoolSize = 128

type Renderer struct {
	log logger.Logger

	bufPool   *bpool.BufferPool
	templates Templates
	Version   string
	BaseURL   string
}

func NewRenderer(log logger.Logger, templates Templates, baseURL string, version string) *Renderer {
	return &Renderer{
		log:       log.WithNamespaceAppended("renderer"),
		bufPool:   bpool.NewBufferPool(defaultBufferPoolSize),
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

		log := tr.log.WithCtx(logger.Ctx{
			"template_name": templateName,
		})

		template, ok := tr.templates.Get(templateName)
		if !ok {
			log.Error("Template not found", nil, nil)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err != nil {
			log.Error("Render", errors.Trace(err), nil)
			w.WriteHeader(http.StatusInternalServerError)
		}

		dataMap := map[string]interface{}{
			"Data":    data,
			"BaseURL": tr.BaseURL,
			"Version": tr.Version,
		}

		buf := tr.bufPool.Get()
		defer tr.bufPool.Put(buf)

		log.Trace("Rendering template", nil)

		err = template.Execute(buf, dataMap)
		if err != nil {
			log.Error("Execute template", errors.Trace(err), nil)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)

		if _, err := buf.WriteTo(w); err != nil {
			log.Error("Write buffer", errors.Trace(err), nil)
		}
	}

	return http.HandlerFunc(fn)
}
