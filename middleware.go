package main

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/handlers"
)

func timerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rCtx := r.Context()
		newReq := r.WithContext(context.WithValue(rCtx, "start", time.Now()))
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		next.ServeHTTP(w, newReq)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	// Headers as listed by the Stremio example addon.
	//
	// According to logs an actual stream request sends these headers though:
	//   Header:map[
	// 	  Accept:[*/*]
	// 	  Accept-Encoding:[gzip, deflate, br]
	// 	  Connection:[keep-alive]
	// 	  Origin:[https://app.strem.io]
	// 	  User-Agent:[Mozilla/5.0 (Windows NT 6.2; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) QtWebEngine/5.9.9 Chrome/56.0.2924.122 Safari/537.36 StremioShell/4.4.106]
	// ]
	headersOk := handlers.AllowedHeaders([]string{
		"Accept",
		"Accept-Language",
		"Content-Type",
		"Origin", // Not "safelisted" in the specification

		// Non-default for gorilla/handlers CORS handling
		"Accept-Encoding",
		"Content-Language", // "Safelisted" in the specification
		"X-Requested-With",
	})
	originsOk := handlers.AllowedOrigins([]string{"*"})
	methodsOk := handlers.AllowedMethods([]string{"GET"})
	return handlers.CORS(originsOk, headersOk, methodsOk)(next)
}

var recoveryMiddleware = handlers.RecoveryHandler(handlers.PrintRecoveryStack(true))

func loggingMiddleware(before http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rCtx := r.Context()
		// First call the *before* handler!
		before.ServeHTTP(w, r)
		// Then log
		reqStart := rCtx.Value("start").(time.Time)
		duration := time.Since(reqStart).Milliseconds()
		durationString := strconv.FormatInt(duration, 10) + "ms"

		log.Printf("Handled request. Method: %v; URL: %v; Remote addr: %v; User-Agent: %v; Duration: %v\n", r.Method, r.URL, r.RemoteAddr, r.Header.Get("User-Agent"), durationString)
	})
}
