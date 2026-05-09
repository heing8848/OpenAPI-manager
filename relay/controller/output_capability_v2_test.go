package controller

import (
	"testing"

	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

func TestGetStandaloneOutputCapabilityV2ImageCoverage(t *testing.T) {
	testCases := []struct {
		name        string
		channelType int
		want        bool
	}{
		{name: "openai compatible image supported", channelType: channeltype.OpenAICompatible, want: true},
		{name: "azure image supported", channelType: channeltype.Azure, want: true},
		{name: "replicate image supported", channelType: channeltype.Replicate, want: true},
		{name: "zhipu image supported", channelType: channeltype.Zhipu, want: true},
		{name: "gemini image unsupported", channelType: channeltype.Gemini, want: false},
		{name: "groq image unsupported", channelType: channeltype.Groq, want: false},
		{name: "anthropic image unsupported", channelType: channeltype.Anthropic, want: false},
		{name: "baidu image unsupported", channelType: channeltype.Baidu, want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := GetStandaloneOutputCapabilityV2(tc.channelType).ImageGeneration
			if got != tc.want {
				t.Fatalf("ImageGeneration mismatch, got %v want %v", got, tc.want)
			}
		})
	}
}

func TestGetStandaloneOutputCapabilityV2AudioCoverage(t *testing.T) {
	testCases := []struct {
		name        string
		channelType int
		want        bool
	}{
		{name: "openai audio supported", channelType: channeltype.OpenAI, want: true},
		{name: "azure audio supported", channelType: channeltype.Azure, want: true},
		{name: "custom audio supported", channelType: channeltype.Custom, want: true},
		{name: "openai compatible audio supported", channelType: channeltype.OpenAICompatible, want: true},
		{name: "zhipu audio unsupported", channelType: channeltype.Zhipu, want: false},
		{name: "ali audio unsupported", channelType: channeltype.Ali, want: false},
		{name: "replicate audio unsupported", channelType: channeltype.Replicate, want: false},
		{name: "groq audio unsupported", channelType: channeltype.Groq, want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := GetStandaloneOutputCapabilityV2(tc.channelType).AudioSpeech
			if got != tc.want {
				t.Fatalf("AudioSpeech mismatch, got %v want %v", got, tc.want)
			}
		})
	}
}

func TestValidateStandaloneOutputCapabilityV2(t *testing.T) {
	imageMeta := &meta.Meta{ChannelType: channeltype.Gemini, ChannelId: 7, ActualModelName: "gemini-2.5-flash"}
	imageErr := ValidateStandaloneOutputCapabilityV2(imageMeta, relaymode.ImagesGenerations)
	if imageErr == nil {
		t.Fatalf("expected unsupported image capability error")
	}
	if imageErr.Code != "image_generation_unsupported_v2" {
		t.Fatalf("unexpected image error code: %v", imageErr.Code)
	}

	audioMeta := &meta.Meta{ChannelType: channeltype.OpenAICompatible, ChannelId: 9, ActualModelName: "tts-1"}
	audioErr := ValidateStandaloneOutputCapabilityV2(audioMeta, relaymode.AudioSpeech)
	if audioErr != nil {
		t.Fatalf("expected audio capability to be supported, got %+v", audioErr)
	}
}
