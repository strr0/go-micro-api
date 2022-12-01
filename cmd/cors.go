package cmd

import "net/http"

func CorsMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Access-Control-Allow-Origin", "*")
		writer.Header().Set("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token,Authorization,Token,X-Token,X-User-Id")
		writer.Header().Set("Access-Control-Allow-Methods", "POST,GET,OPTIONS,DELETE,PUT")
		writer.Header().Set("Access-Control-Expose-Headers", "Content-Length,Access-Control-Allow-Origin,Access-Control-Allow-Headers,Content-Type")
		writer.Header().Set("Access-Control-Allow-Credentials", "true")
		if request.Method == http.MethodOptions {
			writer.WriteHeader(http.StatusOK)
			return
		}
		handler.ServeHTTP(writer, request)
	})
}
