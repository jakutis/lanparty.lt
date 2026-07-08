// Command api is the lanparty.lt HTTP API server.
package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

// config holds all runtime configuration, sourced from the environment.
type config struct {
	port    string
	apiKey  string
	baseURL string
	model   string
}

func main() {
	cfg := loadConfig()
	gen := newLLMGenerator(cfg)

	mux := http.NewServeMux()
	registerRoutes(mux, gen)

	srv := &http.Server{
		Addr:              ":" + cfg.port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      5 * time.Minute,
	}

	log.Printf("api listening on :%s", cfg.port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func loadConfig() config {
	cfg := config{
		port:    envDefault("PORT", "8080"),
		apiKey:  os.Getenv("OPENROUTER_API_KEY"),
		baseURL: envDefault("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1"),
		model:   os.Getenv("OPENROUTER_MODEL"),
	}
	log.Printf("config: port=%s base-url=%s model=%s api-key-set=%v",
		cfg.port, cfg.baseURL, cfg.model, cfg.apiKey != "")
	return cfg
}

func envDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// registerRoutes mounts every endpoint under the "/v1" prefix required by the
// API specification.
func registerRoutes(mux *http.ServeMux, gen Generator) {
	v1 := http.NewServeMux()
	v1.HandleFunc("POST /representation", representationHandler(gen))
	mux.Handle("/v1/", http.StripPrefix("/v1", v1))
}
