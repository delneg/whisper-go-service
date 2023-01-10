package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	// Package imports
	whisper "github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
	wav "github.com/go-audio/wav"
)

func WhisperProcess(model whisper.Model, path string, lang string, speedup, tokens bool) (string, error) {
	var data []float32

	// Create processing context
	context, err := model.NewContext()
	if err != nil {
		return "", err
	}

	// Open the file
	fh, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer fh.Close()

	// Decode the WAV file
	dec := wav.NewDecoder(fh)
	if buf, err := dec.FullPCMBuffer(); err != nil {
		return "", err
	} else if dec.SampleRate != whisper.SampleRate {
		return "", fmt.Errorf("unsupported sample rate: %d", dec.SampleRate)
	} else if dec.NumChans != 1 {
		return "", fmt.Errorf("unsupported number of channels: %d", dec.NumChans)
	} else {
		data = buf.AsFloat32Buffer().Data
	}

	// Set the parameters
	var cb whisper.SegmentCallback
	if lang != "" {
		if err := context.SetLanguage(lang); err != nil {
			return "", err
		}
	}
	context.SetTranslate(false)
	if speedup {
		context.SetSpeedup(true)
	}
	if tokens {
		cb = func(segment whisper.Segment) {
			fmt.Printf("%02d [%6s->%6s] ", segment.Num, segment.Start.Truncate(time.Millisecond), segment.End.Truncate(time.Millisecond))
			for _, token := range segment.Tokens {
				fmt.Printf("%q ", token.Text)
			}
			fmt.Println("")
		}
	}

	// Process the data
	if err := context.Process(data, cb); err != nil {
		return "", err
	}

	var resultString strings.Builder
	// Print out the results
	for {
		segment, err := context.NextSegment()
		if err == io.EOF {
			break
		} else if err != nil {
			return "", err
		}
		_, err = fmt.Fprintf(&resultString, "%s\n", segment.Text)
		//_, err = fmt.Fprintf(&resultString, "[%6s->%6s] %s\n", segment.Start.Truncate(time.Millisecond), segment.End.Truncate(time.Millisecond), segment.Text)
		if err != nil {
			log.Printf("Error printing to stringBuilder - %v", err)
		}
	}

	// Return success
	return resultString.String(), nil
}
