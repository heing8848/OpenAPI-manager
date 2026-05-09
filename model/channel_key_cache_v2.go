package model

import (
	"sync"
	"time"

	"github.com/songquanpeng/one-api/common/config"
	"gorm.io/gorm"
)

type channelKeyCacheEntryV2 struct {
	keys      []ChannelKey
	expiresAt time.Time
}

var (
	channelKeyCacheV2           sync.Map
	channelKeySuccessStateV2    sync.Map
	channelKeyCacheDisabledV2   = config.ChannelKeyCacheTTL <= 0
	channelKeySuccessSyncWindow = time.Duration(config.ChannelKeySuccessSyncInterval) * time.Second
)

func cloneChannelKeysV2(keys []ChannelKey) []ChannelKey {
	if len(keys) == 0 {
		return nil
	}
	cloned := make([]ChannelKey, len(keys))
	copy(cloned, keys)
	return cloned
}

func getChannelKeyCacheTTL() time.Duration {
	return time.Duration(config.ChannelKeyCacheTTL) * time.Second
}

func GetChannelKeysByChannelIdCachedV2(channelId int) ([]ChannelKey, error) {
	if !channelKeyCacheDisabledV2 {
		if cachedValue, ok := channelKeyCacheV2.Load(channelId); ok {
			entry := cachedValue.(channelKeyCacheEntryV2)
			if time.Now().Before(entry.expiresAt) {
				return cloneChannelKeysV2(entry.keys), nil
			}
			channelKeyCacheV2.Delete(channelId)
		}
	}

	keys, err := GetChannelKeysByChannelId(channelId)
	if err != nil {
		return nil, err
	}
	SetChannelKeysCacheV2(channelId, keys)
	return keys, nil
}

func SetChannelKeysCacheV2(channelId int, keys []ChannelKey) {
	if channelKeyCacheDisabledV2 {
		return
	}
	channelKeyCacheV2.Store(channelId, channelKeyCacheEntryV2{
		keys:      cloneChannelKeysV2(keys),
		expiresAt: time.Now().Add(getChannelKeyCacheTTL()),
	})
}

func InvalidateChannelKeysCacheV2(channelId int) {
	channelKeyCacheV2.Delete(channelId)
}

func UpdateChannelKeySuccessCacheV2(channelId int, channelKeyId int, now time.Time) {
	if channelKeyCacheDisabledV2 {
		return
	}
	cachedValue, ok := channelKeyCacheV2.Load(channelId)
	if !ok {
		return
	}
	entry := cachedValue.(channelKeyCacheEntryV2)
	keys := cloneChannelKeysV2(entry.keys)
	for i := range keys {
		if keys[i].Id != channelKeyId {
			continue
		}
		keys[i].Status = ChannelKeyStatusEnabled
		keys[i].FailCount = 0
		keys[i].LastError = ""
		keys[i].CooldownUntil = nil
		keys[i].LastUsedAt = &now
		break
	}
	SetChannelKeysCacheV2(channelId, keys)
}

func ShouldPersistChannelKeySuccessV2(channelKeyId int) bool {
	if channelKeySuccessSyncWindow <= 0 {
		return true
	}
	now := time.Now()
	lastValue, loaded := channelKeySuccessStateV2.Load(channelKeyId)
	if loaded {
		lastTime := lastValue.(time.Time)
		if now.Sub(lastTime) < channelKeySuccessSyncWindow {
			return false
		}
	}
	channelKeySuccessStateV2.Store(channelKeyId, now)
	return true
}

func MarkChannelKeySuccessV2(channelId int, channelKeyId int) error {
	if channelKeyId == 0 {
		return nil
	}
	now := time.Now()
	UpdateChannelKeySuccessCacheV2(channelId, channelKeyId, now)
	if !ShouldPersistChannelKeySuccessV2(channelKeyId) {
		return nil
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&ChannelKey{}).Where("channel_id = ? AND id = ?", channelId, channelKeyId).Updates(map[string]any{
			"status":         ChannelKeyStatusEnabled,
			"fail_count":     0,
			"last_error":     "",
			"last_used_at":   now,
			"cooldown_until": nil,
		}).Error; err != nil {
			return err
		}
		return syncChannelKeyMirrorTx(tx, &Channel{Id: channelId})
	})
}
