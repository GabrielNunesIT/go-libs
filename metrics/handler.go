package metrics

import "net/http"

// Handler returns an http.Handler that serves the metrics from the given
// Registry in Prometheus exposition format. This is a convenience wrapper
// around Registry.Handler() for use as a standalone function.
func Handler(reg *Registry) http.Handler {
	return reg.Handler()
}
