package app

import (
	"log"
	"net/http"
	"proxy/internal/env"
	"proxy/internal/handler"
)

type App struct{}

const (
	successCode = 0
)

func New() *App {
	return &App{}
}

func (a *App) Run() (exitCode int) {

	env.LoadEnv()

	mux := http.NewServeMux()

	port := env.GetEnv("PORT", "8080")

	proxyHandel := handler.New()

	mux.HandleFunc("/api/v1/measure", proxyHandel.Handle)

	log.Fatal(http.ListenAndServe(":"+port, mux))

	return successCode
}
