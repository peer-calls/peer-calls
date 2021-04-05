package logger

// Ctx is the logging context for each logging entry.
type Ctx map[string]interface{}

// WithCtx returns a new context which is a result of a merge of the current
// and the new context. The current context is not modified.
func (c Ctx) WithCtx(newCtx Ctx) Ctx {
	if c == nil {
		return newCtx
	}

	if newCtx == nil {
		return c
	}

	ret := make(Ctx, len(c)+len(newCtx))

	for k, v := range c {
		ret[k] = v
	}

	for k, v := range newCtx {
		ret[k] = v
	}

	return ret
}
