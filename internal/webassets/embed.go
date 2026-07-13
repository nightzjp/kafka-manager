package webassets

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed fallback/* dist-built/*
var embedded embed.FS

func Handler(api http.Handler) http.Handler {
	sub, err := fs.Sub(embedded, "dist-built")
	if err != nil {
		panic(err)
	}
	if _, err := fs.ReadFile(sub, "index.html"); err != nil {
		sub, err = fs.Sub(embedded, "fallback")
		if err != nil {
			panic(err)
		}
	}
	return NewHandler(api, sub)
}
