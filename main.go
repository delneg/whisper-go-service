package main

import (
	"fmt"
	"github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
	ffmpeg "github.com/u2takey/ffmpeg-go"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	tele "gopkg.in/telebot.v3"
)

var WhisperModel string
var KeepFiles string
var BotToken string

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
	BotToken = os.Getenv("BOT_TOKEN")
	if BotToken == "" {
		log.Printf("No BOT_TOKEN ENV found. Trying to get .env file.")
		err := godotenv.Load()
		if err != nil {
			log.Printf("No .env file found... Defaulting BOT_TOKEN to ")
			BotToken = ""
		}
		os.Getenv("BOT_TOKEN")
	}
}

func main() {
	// Get the environment variables
	setEnvVariables()

	pref := tele.Settings{
		Token:  BotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}
	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return
	}

	// Set up the router and routes
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(JSONMiddleware)

	b.Handle(tele.OnVoice, func(c tele.Context) error {
		voice := c.Message().Voice
		log.Printf("Received voice %v, on disk %v", voice.FileID, voice.OnDisk())
		tempFile, err := os.CreateTemp("", "voice_*.ogg")

		if err != nil {
			return c.Send(fmt.Sprintf("Can't create temp file %v", err))
		}
		defer func(name string) {
			err := os.Remove(name)
			if err != nil {
				log.Fatalf("Error removing temp file %v", err)
			}
		}(tempFile.Name())

		model, err := whisper.New(WhisperModel)
		if err != nil {
			//log.Fatal(err)
			return c.Send(fmt.Sprintf("Error while loading the model - %v", err))
		}
		defer func(model whisper.Model) {
			err := model.Close()
			if err != nil {
				log.Fatal(err)
			}
		}(model)

		err = b.Download(&voice.File, tempFile.Name())
		if err != nil {
			return c.Send(fmt.Sprintf("Can't download file %v", err))
		}
		log.Printf("Downloaded voice %s", tempFile.Name())

		ffmpegArgs := make([]ffmpeg.KwArgs, 0)

		// Append all args and merge to single KwArgs
		ffmpegArgs = append(ffmpegArgs, ffmpeg.KwArgs{"ar": 16000, "ac": 1, "c:a": "pcm_s16le"})
		args := ffmpeg.MergeKwArgs(ffmpegArgs)

		outputTempFile, err := os.CreateTemp("", "voice_*.wav")

		if err != nil {
			return c.Send(fmt.Sprintf("Can't create output temp file %v", err))
		}
		defer func(name string) {
			err := os.Remove(name)
			if err != nil {
				log.Fatalf("Error removing output temp file %v", err)
			}
		}(outputTempFile.Name())

		err = ffmpeg.Input(tempFile.Name()).
			Output(outputTempFile.Name(), args).
			OverWriteOutput().ErrorToStdOut().Run()

		if err != nil {
			log.Printf("ffmpeg Err: %v", err)
			return c.Send(fmt.Sprintf("Error while encoding to wav: %v", err))
		}

		resultingText, err := WhisperProcess(model, outputTempFile.Name(), "", false, false)

		if err != nil {
			log.Printf("Whisper Error: %v", err)
			return c.Send(fmt.Sprintf("Whisper error: %v", err))
		}
		return c.Send(resultingText)
	})

	b.Start()

	//rootHandler := RootHandler{Model: model}
	//r.Post("/transcribe", rootHandler.transcribe)
	//r.Get("/getsubs", rootHandler.getSubsFile)
	//
	//c := cors.New(cors.Options{
	//	AllowedOrigins: []string{"*"},
	//	AllowedMethods: []string{http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodPatch},
	//})
	//
	//handler := c.Handler(r)
	//log.Printf("Starting backend server at :9090...")
	//err = http.ListenAndServe(":9090", handler)
	//if err != nil {
	//	log.Fatal(err)
	//}
}

func JSONMiddleware(hndlr http.Handler) http.Handler {
	// This function sets the response content type to json.
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		hndlr.ServeHTTP(w, r)
	})
}
