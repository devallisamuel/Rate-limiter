package limiter

import (
	"net"
	"net/http"
)

// func (l *Limiter) Middleware(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		if !l.Allow() {
// 			http.Error(w, "too many requests", http.StatusTooManyRequests)
// 			return
// 		}
// 		next.ServeHTTP(w, r)
// 	})
// }

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (s *Store) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := s.Get(clientIP(r))
		if !l.Allow() {
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
