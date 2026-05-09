package model

import (
	"testing"
	"time"
)

func TestCollectRoundRobinChannelKeyCandidatesV2ReactivatesCooldownKeys(t *testing.T) {
	cooldownUntil := time.Now().Add(10 * time.Minute)
	keys := []ChannelKey{
		{Id: 1, KeyValue: "key-1", Position: 0, Status: ChannelKeyStatusCooldown, CooldownUntil: &cooldownUntil},
		{Id: 2, KeyValue: "key-2", Position: 1, Status: ChannelKeyStatusEnabled},
		{Id: 3, KeyValue: "key-3", Position: 2, Status: ChannelKeyStatusDisabled},
	}

	candidates, reactivatedIds := collectRoundRobinChannelKeyCandidatesV2(keys)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 active candidates, got %d", len(candidates))
	}
	if len(reactivatedIds) != 1 || reactivatedIds[0] != 1 {
		t.Fatalf("expected cooldown key #1 to be reactivated, got %#v", reactivatedIds)
	}
	if candidates[0].Status != ChannelKeyStatusEnabled || candidates[0].CooldownUntil != nil {
		t.Fatalf("expected first candidate to be enabled with no cooldown, got %+v", candidates[0])
	}
}

func TestApplyChannelKeyFailureStateV2Keeps404KeyEnabled(t *testing.T) {
	key := ChannelKey{Id: 1, Status: ChannelKeyStatusEnabled, FailCount: 0}
	next := applyChannelKeyFailureStateV2(key, classifyChannelKeyFailureV2(404, "model not found"), "model not found", time.Now())

	if next.Status != ChannelKeyStatusEnabled {
		t.Fatalf("expected key to stay enabled after 404, got %s", next.Status)
	}
	if next.CooldownUntil != nil {
		t.Fatalf("expected no cooldown after 404, got %v", next.CooldownUntil)
	}
	if next.FailCount != 1 {
		t.Fatalf("expected fail count to increment, got %d", next.FailCount)
	}
}

func TestApplyChannelKeyFailureStateV2DisablesUnauthorizedKey(t *testing.T) {
	key := ChannelKey{Id: 1, Status: ChannelKeyStatusEnabled, FailCount: 0}
	next := applyChannelKeyFailureStateV2(key, classifyChannelKeyFailureV2(401, "invalid api key"), "invalid api key", time.Now())

	if next.Status != ChannelKeyStatusDisabled {
		t.Fatalf("expected key to be disabled after invalid auth, got %s", next.Status)
	}
	if next.CooldownUntil != nil {
		t.Fatalf("expected no cooldown timestamp for disabled key, got %v", next.CooldownUntil)
	}
}

func TestApplyChannelKeyFailureStateV2KeepsPermission403KeyEnabled(t *testing.T) {
	key := ChannelKey{Id: 1, Status: ChannelKeyStatusEnabled, FailCount: 0}
	next := applyChannelKeyFailureStateV2(
		key,
		classifyChannelKeyFailureV2(403, "The model `openai/gpt-oss-120b` is blocked at the organization level"),
		"The model `openai/gpt-oss-120b` is blocked at the organization level",
		time.Now(),
	)

	if next.Status != ChannelKeyStatusEnabled {
		t.Fatalf("expected 403 permission errors to keep key enabled, got %s", next.Status)
	}
	if next.CooldownUntil != nil {
		t.Fatalf("expected no cooldown for permission 403, got %v", next.CooldownUntil)
	}
}

func TestApplyChannelKeyFailureStateV2StillDisables403CredentialErrors(t *testing.T) {
	key := ChannelKey{Id: 1, Status: ChannelKeyStatusEnabled, FailCount: 0}
	next := applyChannelKeyFailureStateV2(
		key,
		classifyChannelKeyFailureV2(403, "invalid api key"),
		"invalid api key",
		time.Now(),
	)

	if next.Status != ChannelKeyStatusDisabled {
		t.Fatalf("expected credential 403 to disable key, got %s", next.Status)
	}
}

func TestApplyChannelKeyFailureStateV2KeepsCloudflareChallengeEnabled(t *testing.T) {
	key := ChannelKey{Id: 1, Status: ChannelKeyStatusEnabled, FailCount: 0}
	next := applyChannelKeyFailureStateV2(
		key,
		classifyChannelKeyFailureV2(403, "Just a moment... Enable JavaScript and cookies to continue"),
		"Just a moment... Enable JavaScript and cookies to continue",
		time.Now(),
	)

	if next.Status != ChannelKeyStatusEnabled {
		t.Fatalf("expected Cloudflare challenge 403 to keep key enabled, got %s", next.Status)
	}
	if next.CooldownUntil != nil {
		t.Fatalf("expected no cooldown for Cloudflare challenge, got %v", next.CooldownUntil)
	}
}
