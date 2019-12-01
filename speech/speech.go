package speech

import (
	"context"
	"fmt"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"github.com/jchorl/gowaker/requestcontext"
	"github.com/valyala/fasthttp"
	"google.golang.org/api/option"
	texttospeechpb "google.golang.org/genproto/googleapis/cloud/texttospeech/v1"
)

// New returns a new text-to-speech client
func New(credentialFile string) (*texttospeech.Client, error) {
	ctx := context.Background()

	client, err := texttospeech.NewClient(ctx, option.WithCredentialsFile(credentialFile))
	if err != nil {
		return nil, fmt.Errorf("creating text-to-speech client: %w", err)
	}

	return client, nil
}

// GetAudioContent will translate a string into audio
func GetAudioContent(ctx *fasthttp.RequestCtx, text string) ([]byte, error) {
	client := requestcontext.Speech(ctx)

	// Perform the text-to-speech request on the text input with the selected
	// voice parameters and audio file type.
	req := texttospeechpb.SynthesizeSpeechRequest{
		// Set the text input to be synthesized.
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{Text: text},
		},
		// Build the voice request, select the language code ("en-US") and the SSML
		// voice gender ("neutral").
		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: "en-US",
			SsmlGender:   texttospeechpb.SsmlVoiceGender_NEUTRAL,
		},
		// Select the type of audio file you want returned.
		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding: texttospeechpb.AudioEncoding_LINEAR16,
		},
	}

	resp, err := client.SynthesizeSpeech(context.TODO(), &req)
	if err != nil {
		return nil, fmt.Errorf("synthesizing speech: %w", err)
	}

	// The resp's AudioContent is binary.
	return resp.AudioContent, nil
}
