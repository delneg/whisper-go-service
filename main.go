package main

import (
	"github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

var WhisperModel string
var KeepFiles string

func setEnvVariables() {

	WhisperModel = os.Getenv("WHISPER_MODEL")
	if WhisperModel == "" {
		log.Printf("No WHISPER_MODEL ENV found. Trying to get .env file.")
		err := godotenv.Load()
		if err != nil {
			log.Printf("No .env file found... Defaulting WHISPER_MODEL to 0")
			WhisperModel = "small"
		}
		os.Getenv("WHISPER_MODEL")
		if WhisperModel == "" {
			WhisperModel = "models/ggml-medium.bin"
		}
	}
	log.Printf("Selected model: %v", WhisperModel)

	KeepFiles = os.Getenv("KEEP_FILES")
	if KeepFiles == "" {
		log.Printf("No KEEP_FILES ENV found. Trying to get .env file.")
		err := godotenv.Load()
		if err != nil {
			log.Printf("No .env file found... Defaulting KEEP_FILES to false")
			KeepFiles = "false"
		}
		os.Getenv("KEEP_FILES")
		if KeepFiles == "" {
			KeepFiles = "false"
		}
	}
}

func main() {
	// Get the environment variables
	setEnvVariables()

	// Set up the router and routes
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(JSONMiddleware)

	model, err := whisper.New(WhisperModel)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer model.Close()

	rootHandler := RootHandler{Model: model}
	r.Post("/transcribe", rootHandler.transcribe)
	r.Get("/getsubs", rootHandler.getSubsFile)

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodPatch},
	})

	handler := c.Handler(r)
	log.Printf("Starting backend server at :9090...")
	err = http.ListenAndServe(":9090", handler)
	if err != nil {
		log.Fatal(err)
	}
}

func JSONMiddleware(hndlr http.Handler) http.Handler {
	// This function sets the response content type to json.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		hndlr.ServeHTTP(w, r)
	})
}
