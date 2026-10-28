package consolepages

import "net/http"

func mountedAt(pathPrefix string, console http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle(pathPrefix+"/", http.StripPrefix(pathPrefix, http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			console.ServeHTTP(mountedResponseWriter{ResponseWriter: w, pathPrefix: pathPrefix}, r)
		},
	)))
	return mux
}

type mountedResponseWriter struct {
	http.ResponseWriter
	pathPrefix string
}

func (w mountedResponseWriter) WriteHeader(status int) {
	location := w.Header().Get("Location")
	if isLocalPath(location) {
		w.Header().Set("Location", w.pathPrefix+location)
	}
	w.ResponseWriter.WriteHeader(status)
}
