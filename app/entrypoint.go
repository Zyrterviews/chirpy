package app

import "net/http"

func GetStaticAssets() http.Handler {
	return http.StripPrefix("/app/", http.FileServer(http.Dir(".")))
}
