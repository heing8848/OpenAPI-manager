package model

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestPrepareChannelModelIDPrefixForWriteV2LeavesModelsUnchangedWithoutPrefix(t *testing.T) {
	emptyPrefix := ""
	channel := &Channel{
		Models:        "moonshotai/kimi-k2.5,openai/gpt-oss-120b,google/gemma-4-31b-it",
		ModelIDPrefix: &emptyPrefix,
	}

	PrepareChannelModelIDPrefixForWriteV2(channel, nil)

	wantModels := []string{
		"moonshotai/kimi-k2.5",
		"openai/gpt-oss-120b",
		"google/gemma-4-31b-it",
	}
	if !equalModelSetsV2(channel.Models, wantModels) {
		t.Fatalf("expected models %#v, got %q", wantModels, channel.Models)
	}
	if channel.GetModelIDPrefix() != "" {
		t.Fatalf("expected empty prefix, got %q", channel.GetModelIDPrefix())
	}
	if mapping := channel.GetModelMappingV2(); mapping != nil {
		t.Fatalf("expected no auto model mapping, got %#v", mapping)
	}
}

func TestPrepareChannelModelIDPrefixForWriteV2AppliesPrefixToAllModels(t *testing.T) {
	prefix := "nvidia"
	channel := &Channel{
		Models:        "moonshotai/kimi-k2.5,openai/gpt-oss-120b,google/gemma-4-31b-it",
		ModelIDPrefix: &prefix,
	}

	PrepareChannelModelIDPrefixForWriteV2(channel, nil)

	wantModels := []string{
		"nvidia-moonshotai/kimi-k2.5",
		"nvidia-openai/gpt-oss-120b",
		"nvidia-google/gemma-4-31b-it",
	}
	if !equalModelSetsV2(channel.Models, wantModels) {
		t.Fatalf("expected models %#v, got %q", wantModels, channel.Models)
	}

	wantMapping := map[string]string{
		"nvidia-moonshotai/kimi-k2.5":  "moonshotai/kimi-k2.5",
		"nvidia-openai/gpt-oss-120b":   "openai/gpt-oss-120b",
		"nvidia-google/gemma-4-31b-it": "google/gemma-4-31b-it",
	}
	if mapping := channel.GetModelMappingV2(); !reflect.DeepEqual(mapping, wantMapping) {
		t.Fatalf("expected mapping %#v, got %#v", wantMapping, mapping)
	}
}

func TestPrepareChannelModelIDPrefixForWriteV2ClearsExistingPrefix(t *testing.T) {
	previousPrefix := "nvidia"
	nextPrefix := ""
	previousChannel := &Channel{
		Models:        "nvidia-moonshotai/kimi-k2.5,nvidia-openai/gpt-oss-120b,nvidia-google/gemma-4-31b-it",
		ModelIDPrefix: &previousPrefix,
	}
	channel := &Channel{
		Models:        previousChannel.Models,
		ModelIDPrefix: &nextPrefix,
	}

	PrepareChannelModelIDPrefixForWriteV2(channel, previousChannel)

	wantModels := []string{
		"moonshotai/kimi-k2.5",
		"openai/gpt-oss-120b",
		"google/gemma-4-31b-it",
	}
	if !equalModelSetsV2(channel.Models, wantModels) {
		t.Fatalf("expected models %#v, got %q", wantModels, channel.Models)
	}
	if channel.GetModelIDPrefix() != "" {
		t.Fatalf("expected empty prefix, got %q", channel.GetModelIDPrefix())
	}
	if mapping := channel.GetModelMappingV2(); mapping != nil {
		t.Fatalf("expected no auto model mapping, got %#v", mapping)
	}
}

func equalModelSetsV2(csv string, expected []string) bool {
	got := strings.Split(csv, ",")
	sort.Strings(got)

	want := append([]string(nil), expected...)
	sort.Strings(want)

	return reflect.DeepEqual(got, want)
}
