package main

import (
	"github.com/go-chi/chi/v5"
)

func NewRouter() chi.Router {
	r := chi.NewRouter()
	RegisterCommonMiddlewares(r)
	return r
}
