package httpingress

import (
	"net/http"

	"miren.dev/runtime/api/ingress/ingress_v1alpha"
)

func (s *Server) wafMiddleware(route *ingress_v1alpha.HttpRoute, next http.HandlerFunc) http.HandlerFunc {
	if route.WafLevel <= 0 {
		return next
	}

	return func(w http.ResponseWriter, r *http.Request) {
		handler, err := s.wafEngine.Handler(int(route.WafLevel), http.HandlerFunc(next))
		if err != nil {
			s.Log.Error("failed to create WAF handler", "error", err, "level", route.WafLevel)
			next(w, r)
			return
		}

		handler.ServeHTTP(w, r)
	}
}
