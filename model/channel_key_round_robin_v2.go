package model

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// PrepareChannelKeyCandidatesV2 keeps the old key pool rotation behavior, but
// treats cooldown as a legacy state. In round-robin mode we only skip keys that
// were explicitly disabled, so existing cooldown entries are immediately
// re-enabled and can continue participating in request-level rotation.
func PrepareChannelKeyCandidatesV2(channel *Channel) ([]ChannelKey, error) {
	keys, err := GetChannelKeysByChannelIdCachedV2(channel.Id)
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 && strings.TrimSpace(channel.Key) != "" {
		if err = ensureLegacyChannelKeys(channel); err != nil {
			return nil, err
		}
		keys, err = GetChannelKeysByChannelIdCachedV2(channel.Id)
		if err != nil {
			return nil, err
		}
	}
	if len(keys) == 0 {
		return nil, errors.New("channel has no configured keys")
	}

	candidates, reactivatedIds := collectRoundRobinChannelKeyCandidatesV2(keys)

	if len(reactivatedIds) > 0 {
		err = DB.Model(&ChannelKey{}).Where("id IN ?", reactivatedIds).Updates(map[string]any{
			"status":         ChannelKeyStatusEnabled,
			"cooldown_until": nil,
		}).Error
		if err != nil {
			return nil, err
		}
		if err = syncChannelKeyMirror(channel.Id); err != nil {
			return nil, err
		}
	}

	if len(candidates) == 0 {
		return nil, errors.New("channel has no available keys")
	}

	channel.Keys = keys
	start := nextChannelKeyCursor(channel.Id, len(candidates))
	if start == 0 {
		return candidates, nil
	}
	return append(candidates[start:], candidates[:start]...), nil
}

// MarkChannelKeyFailureV2 keeps request rotation simple: only clearly invalid
// keys are disabled. Other upstream failures are recorded on the key, but do
// not trigger cooldown, so the next request can continue rotating to the next
// key instead of exhausting the whole pool inside one request.
func MarkChannelKeyFailureV2(channelId int, channelKeyId int, statusCode int, errMessage string) error {
	if channelKeyId == 0 {
		return nil
	}
	errMessage = truncateChannelKeyError(errMessage)
	action := classifyChannelKeyFailureV2(statusCode, errMessage)
	now := time.Now()

	return DB.Transaction(func(tx *gorm.DB) error {
		var key ChannelKey
		if err := tx.Where("channel_id = ? AND id = ?", channelId, channelKeyId).First(&key).Error; err != nil {
			return err
		}

		nextKeyState := applyChannelKeyFailureStateV2(key, action, errMessage, now)
		updates := map[string]any{
			"last_error":     nextKeyState.LastError,
			"last_used_at":   nextKeyState.LastUsedAt,
			"cooldown_until": nextKeyState.CooldownUntil,
			"fail_count":     nextKeyState.FailCount,
			"status":         nextKeyState.Status,
		}

		if err := tx.Model(&ChannelKey{}).Where("id = ?", channelKeyId).Updates(updates).Error; err != nil {
			return err
		}
		return syncChannelKeyMirrorTx(tx, &Channel{Id: channelId})
	})
}

func classifyChannelKeyFailureV2(statusCode int, errMessage string) channelKeyFailureAction {
	lowerMessage := strings.ToLower(errMessage)

	if isUpstreamCloudflareChallengeMessageV2(lowerMessage) {
		return channelKeyFailureRetry
	}

	if statusCode == 401 || containsAny(lowerMessage,
		"invalid api key",
		"invalid key",
		"api key not valid",
		"api key expired",
		"authentication",
		"unauthorized",
		"account deactivated",
		"account disabled",
		"organization has been disabled",
		"organization has been restricted",
		"your credit balance is too low",
		"insufficient_quota",
		"insufficient quota",
		"credit balance",
		"billing",
		"余额不足",
	) {
		return channelKeyFailureDisable
	}

	return channelKeyFailureRetry
}

func isUpstreamCloudflareChallengeMessageV2(lowerMessage string) bool {
	return containsAny(lowerMessage,
		"enable javascript and cookies to continue",
		"window._cf_chl_opt",
		"__cf_chl_tk",
		"just a moment...",
	)
}

func collectRoundRobinChannelKeyCandidatesV2(keys []ChannelKey) ([]ChannelKey, []int) {
	reactivatedIds := make([]int, 0)
	candidates := make([]ChannelKey, 0, len(keys))
	for _, key := range keys {
		if key.Status == ChannelKeyStatusDisabled {
			continue
		}
		if key.Status == ChannelKeyStatusCooldown {
			reactivatedIds = append(reactivatedIds, key.Id)
			key.Status = ChannelKeyStatusEnabled
			key.CooldownUntil = nil
		}
		candidates = append(candidates, key)
	}
	return candidates, reactivatedIds
}

func applyChannelKeyFailureStateV2(
	key ChannelKey,
	action channelKeyFailureAction,
	errMessage string,
	now time.Time,
) ChannelKey {
	key.LastError = truncateChannelKeyError(errMessage)
	key.FailCount++
	key.CooldownUntil = nil
	key.LastUsedAt = &now

	if action == channelKeyFailureDisable {
		key.Status = ChannelKeyStatusDisabled
		return key
	}

	key.Status = ChannelKeyStatusEnabled
	return key
}
