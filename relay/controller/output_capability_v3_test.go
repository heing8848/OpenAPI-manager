package controller

import (
	"testing"

	"github.com/songquanpeng/one-api/relay/channeltype"
	"github.com/songquanpeng/one-api/relay/meta"
	"github.com/songquanpeng/one-api/relay/relaymode"
)

func TestGetStandaloneOutputCapabilityV3VideoCoverage(t *testing.T) {
	if !GetStandaloneOutputCapabilityV3(channeltype.VideoTaskV1).VideoGeneration {
		t.Fatalf("expected video task channel to support standalone video generation")
	}
	if GetStandaloneOutputCapabilityV3(channeltype.OpenAICompatible).VideoGeneration {
		t.Fatalf("did not expect openai compatible channel to support standalone video generation")
	}
}

func TestValidateStandaloneOutputCapabilityV3(t *testing.T) {
	videoMeta := &meta.Meta{ChannelType: channeltype.VideoTaskV1, ChannelId: 52, ActualModelName: "video-model"}
	if err := ValidateStandaloneOutputCapabilityV3(videoMeta, relaymode.VideosGenerationsV1); err != nil {
		t.Fatalf("expected video generation to be supported, got %+v", err)
	}

	unsupportedMeta := &meta.Meta{ChannelType: channeltype.OpenAICompatible, ChannelId: 50, ActualModelName: "gpt-4o"}
	err := ValidateStandaloneOutputCapabilityV3(unsupportedMeta, relaymode.VideosGenerationsV1)
	if err == nil {
		t.Fatalf("expected unsupported video generation error")
	}
	if err.Code != "video_generation_unsupported_v1" {
		t.Fatalf("unexpected error code: %v", err.Code)
	}
}
