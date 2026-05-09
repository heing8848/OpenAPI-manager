package model

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/logger"
	"gorm.io/gorm"
)

const (
	ChannelKeyStatusEnabled  = "enabled"
	ChannelKeyStatusCooldown = "cooldown"
	ChannelKeyStatusDisabled = "disabled"

	ChannelKeyFailureThreshold = 3
)

var (
	channelKeyShortCooldown = 30 * time.Second
	channelKeyRateCooldown  = 10 * time.Minute
	channelKeyErrorCooldown = 5 * time.Minute
	channelKeyLocalCursor   sync.Map
)

type ChannelKey struct {
	Id            int        `json:"id"`
	ChannelId     int        `json:"channel_id" gorm:"index;not null"`
	KeyValue      string     `json:"key_value" gorm:"type:text;not null"`
	Position      int        `json:"position" gorm:"not null;default:0"`
	Status        string     `json:"status" gorm:"type:varchar(16);not null;default:'enabled';index"`
	FailCount     int        `json:"fail_count" gorm:"not null;default:0"`
	LastError     string     `json:"last_error" gorm:"type:text"`
	LastUsedAt    *time.Time `json:"last_used_at,omitempty"`
	CooldownUntil *time.Time `json:"cooldown_until,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type ChannelWriteRequest struct {
	Id                 int      `json:"id"`
	Type               int      `json:"type"`
	Key                string   `json:"key"`
	Keys               []string `json:"keys"`
	Status             int      `json:"status"`
	Name               string   `json:"name"`
	Weight             *uint    `json:"weight"`
	CreatedTime        int64    `json:"created_time"`
	TestTime           int64    `json:"test_time"`
	ResponseTime       int      `json:"response_time"`
	BaseURL            *string  `json:"base_url"`
	Other              *string  `json:"other"`
	Balance            float64  `json:"balance"`
	BalanceUpdatedTime int64    `json:"balance_updated_time"`
	Models             string   `json:"models"`
	ModelIDPrefix      *string  `json:"model_id_prefix"`
	DisplayOrder       *int64   `json:"display_order"`
	Group              string   `json:"group"`
	UsedQuota          int64    `json:"used_quota"`
	ModelMapping       *string  `json:"model_mapping"`
	Priority           *int64   `json:"priority"`
	Config             string   `json:"config"`
	SystemPrompt       *string  `json:"system_prompt"`
}

type channelKeyFailureAction string

const (
	channelKeyFailureCooldown channelKeyFailureAction = "cooldown"
	channelKeyFailureDisable  channelKeyFailureAction = "disable"
	channelKeyFailureRetry    channelKeyFailureAction = "retry"
)

func (request *ChannelWriteRequest) ToChannel() *Channel {
	return &Channel{
		Id:                 request.Id,
		Type:               request.Type,
		Key:                request.Key,
		Status:             request.Status,
		Name:               request.Name,
		Weight:             request.Weight,
		CreatedTime:        request.CreatedTime,
		TestTime:           request.TestTime,
		ResponseTime:       request.ResponseTime,
		BaseURL:            request.BaseURL,
		Other:              request.Other,
		Balance:            request.Balance,
		BalanceUpdatedTime: request.BalanceUpdatedTime,
		Models:             request.Models,
		ModelIDPrefix:      request.ModelIDPrefix,
		DisplayOrder:       request.DisplayOrder,
		Group:              request.Group,
		UsedQuota:          request.UsedQuota,
		ModelMapping:       request.ModelMapping,
		Priority:           request.Priority,
		Config:             request.Config,
		SystemPrompt:       request.SystemPrompt,
	}
}

func NormalizeChannelKeyValues(rawValues []string) []string {
	values := make([]string, 0, len(rawValues))
	seen := make(map[string]bool)
	for _, rawValue := range rawValues {
		for _, value := range strings.Split(strings.ReplaceAll(rawValue, "\r\n", "\n"), "\n") {
			value = strings.TrimSpace(value)
			if value == "" || seen[value] {
				continue
			}
			seen[value] = true
			values = append(values, value)
		}
	}
	return values
}

func SplitLegacyChannelKeys(raw string) []string {
	if raw == "" {
		return nil
	}
	return NormalizeChannelKeyValues([]string{raw})
}

func (channel *Channel) LoadKeys() error {
	keys, err := GetChannelKeysByChannelIdCachedV2(channel.Id)
	if err != nil {
		return err
	}
	channel.Keys = keys
	return nil
}

func (channel *Channel) ResolvePrimaryKeyValue() string {
	keys := channel.Keys
	if len(keys) == 0 && channel.Id != 0 {
		if loadedKeys, err := GetChannelKeysByChannelIdCachedV2(channel.Id); err == nil {
			keys = loadedKeys
		}
	}
	for _, key := range keys {
		if key.Status != ChannelKeyStatusDisabled {
			return key.KeyValue
		}
	}
	if len(keys) > 0 {
		return keys[0].KeyValue
	}
	return channel.Key
}

func GetChannelKeysByChannelId(channelId int) ([]ChannelKey, error) {
	var keys []ChannelKey
	err := DB.Where("channel_id = ?", channelId).Order("position asc, id asc").Find(&keys).Error
	return keys, err
}

func FillChannelHasDisabledKeys(channels []*Channel) error {
	if len(channels) == 0 {
		return nil
	}
	channelIds := make([]int, 0, len(channels))
	for _, channel := range channels {
		channelIds = append(channelIds, channel.Id)
	}

	var disabledKeys []ChannelKey
	if err := DB.Select("channel_id").Where("channel_id IN ? AND status = ?", channelIds, ChannelKeyStatusDisabled).Group("channel_id").Find(&disabledKeys).Error; err != nil {
		return err
	}

	disabledSet := make(map[int]bool)
	for _, key := range disabledKeys {
		disabledSet[key.ChannelId] = true
	}

	for _, channel := range channels {
		channel.HasDisabledKeys = disabledSet[channel.Id]
	}

	return nil
}

func MigrateChannelKeys() error {
	var channels []Channel
	if err := DB.Where("key <> ''").Find(&channels).Error; err != nil {
		return err
	}
	for i := range channels {
		if err := ensureLegacyChannelKeys(&channels[i]); err != nil {
			return err
		}
	}
	return nil
}

func MigrateChannelKeysV2() error {
	var channels []Channel
	if err := buildLegacyChannelKeyMigrationQueryV2(DB).Find(&channels).Error; err != nil {
		return err
	}
	for i := range channels {
		if err := ensureLegacyChannelKeys(&channels[i]); err != nil {
			return err
		}
	}
	return nil
}

func buildLegacyChannelKeyMigrationQueryV2(db *gorm.DB) *gorm.DB {
	return db.Where("`key` <> ?", "")
}

func ensureLegacyChannelKeys(channel *Channel) error {
	if channel.Id == 0 || strings.TrimSpace(channel.Key) == "" {
		return nil
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		return ensureLegacyChannelKeysTx(tx, channel)
	})
}

func ensureLegacyChannelKeysTx(tx *gorm.DB, channel *Channel) error {
	var count int64
	if err := tx.Model(&ChannelKey{}).Where("channel_id = ?", channel.Id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	keys := SplitLegacyChannelKeys(channel.Key)
	for position, value := range keys {
		if err := tx.Create(&ChannelKey{
			ChannelId: channel.Id,
			KeyValue:  value,
			Position:  position,
			Status:    ChannelKeyStatusEnabled,
		}).Error; err != nil {
			return err
		}
	}
	return syncChannelKeyMirrorTx(tx, channel)
}

func upsertChannelKeysTx(tx *gorm.DB, channel *Channel, keyValues []string) error {
	keyValues = NormalizeChannelKeyValues(keyValues)
	if len(keyValues) == 0 {
		keyValues = SplitLegacyChannelKeys(channel.Key)
	}

	var existingKeys []ChannelKey
	if err := tx.Where("channel_id = ?", channel.Id).Order("position asc, id asc").Find(&existingKeys).Error; err != nil {
		return err
	}

	existingByValue := make(map[string][]ChannelKey)
	for _, key := range existingKeys {
		existingByValue[key.KeyValue] = append(existingByValue[key.KeyValue], key)
	}

	keepIds := make([]int, 0, len(keyValues))
	for position, value := range keyValues {
		existing := existingByValue[value]
		if len(existing) > 0 {
			key := existing[0]
			existingByValue[value] = existing[1:]
			keepIds = append(keepIds, key.Id)
			if err := tx.Model(&ChannelKey{}).Where("id = ?", key.Id).Update("position", position).Error; err != nil {
				return err
			}
			continue
		}

		key := ChannelKey{
			ChannelId: channel.Id,
			KeyValue:  value,
			Position:  position,
			Status:    ChannelKeyStatusEnabled,
		}
		if err := tx.Create(&key).Error; err != nil {
			return err
		}
		keepIds = append(keepIds, key.Id)
	}

	if len(keepIds) == 0 {
		if err := tx.Where("channel_id = ?", channel.Id).Delete(&ChannelKey{}).Error; err != nil {
			return err
		}
	} else {
		if err := tx.Where("channel_id = ? AND id NOT IN ?", channel.Id, keepIds).Delete(&ChannelKey{}).Error; err != nil {
			return err
		}
	}

	if err := syncChannelKeyMirrorTx(tx, channel); err != nil {
		return err
	}
	return nil
}

func syncChannelKeyMirror(channelId int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		return syncChannelKeyMirrorTx(tx, &Channel{Id: channelId})
	})
}

func syncChannelKeyMirrorTx(tx *gorm.DB, channel *Channel) error {
	mirrorKey := ""

	var key ChannelKey
	err := tx.Where("channel_id = ? AND status = ?", channel.Id, ChannelKeyStatusEnabled).
		Order("position asc, id asc").
		First(&key).Error
	if err == nil {
		mirrorKey = key.KeyValue
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	} else {
		err = tx.Where("channel_id = ?", channel.Id).
			Order("position asc, id asc").
			First(&key).Error
		if err == nil {
			mirrorKey = key.KeyValue
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}

	if err := tx.Model(&Channel{}).Where("id = ?", channel.Id).Update("key", mirrorKey).Error; err != nil {
		return err
	}
	channel.Key = mirrorKey
	InvalidateChannelKeysCacheV2(channel.Id)
	return nil
}

func PrepareChannelKeyCandidates(channel *Channel) ([]ChannelKey, error) {
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

	now := time.Now()
	expiredCooldownIds := make([]int, 0)
	candidates := make([]ChannelKey, 0, len(keys))
	for _, key := range keys {
		if key.Status == ChannelKeyStatusDisabled {
			continue
		}
		if key.Status == ChannelKeyStatusCooldown {
			if key.CooldownUntil != nil && key.CooldownUntil.After(now) {
				continue
			}
			expiredCooldownIds = append(expiredCooldownIds, key.Id)
			key.Status = ChannelKeyStatusEnabled
			key.FailCount = 0
			key.CooldownUntil = nil
			key.LastError = ""
		}
		candidates = append(candidates, key)
	}

	if len(expiredCooldownIds) > 0 {
		err = DB.Model(&ChannelKey{}).Where("id IN ?", expiredCooldownIds).Updates(map[string]any{
			"status":         ChannelKeyStatusEnabled,
			"fail_count":     0,
			"last_error":     "",
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

func MarkChannelKeySuccess(channelId int, channelKeyId int) error {
	if channelKeyId == 0 {
		return nil
	}
	now := time.Now()
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

func MarkChannelKeyFailure(channelId int, channelKeyId int, statusCode int, errMessage string) error {
	if channelKeyId == 0 {
		return nil
	}
	errMessage = truncateChannelKeyError(errMessage)
	action := classifyChannelKeyFailure(statusCode, errMessage)
	now := time.Now()

	return DB.Transaction(func(tx *gorm.DB) error {
		var key ChannelKey
		if err := tx.Where("channel_id = ? AND id = ?", channelId, channelKeyId).First(&key).Error; err != nil {
			return err
		}

		updates := map[string]any{
			"last_error": errMessage,
		}

		switch action {
		case channelKeyFailureDisable:
			updates["status"] = ChannelKeyStatusDisabled
			updates["cooldown_until"] = nil
		case channelKeyFailureCooldown:
			updates["status"] = ChannelKeyStatusCooldown
			updates["cooldown_until"] = now.Add(channelKeyRateCooldown)
		default:
			failCount := key.FailCount + 1
			updates["fail_count"] = failCount
			updates["status"] = ChannelKeyStatusCooldown
			if failCount >= ChannelKeyFailureThreshold {
				updates["cooldown_until"] = now.Add(channelKeyErrorCooldown)
			} else {
				updates["cooldown_until"] = now.Add(channelKeyShortCooldown)
			}
		}

		if err := tx.Model(&ChannelKey{}).Where("id = ?", channelKeyId).Updates(updates).Error; err != nil {
			return err
		}
		return syncChannelKeyMirrorTx(tx, &Channel{Id: channelId})
	})
}

func classifyChannelKeyFailure(statusCode int, errMessage string) channelKeyFailureAction {
	lowerMessage := strings.ToLower(errMessage)

	if statusCode == 429 || containsAny(lowerMessage,
		"too many requests",
		"rate limit",
		"resource exhausted",
		"requests per minute",
		"tokens per minute",
		"rpm",
		"tpm",
	) {
		return channelKeyFailureCooldown
	}

	if statusCode == 401 || statusCode == 403 || containsAny(lowerMessage,
		"invalid api key",
		"invalid key",
		"api key not valid",
		"api key expired",
		"authentication",
		"unauthorized",
		"permission denied",
		"permission_error",
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

	if statusCode/100 == 5 || containsAny(lowerMessage,
		"timeout",
		"temporarily unavailable",
		"connection reset",
		"bad gateway",
		"service unavailable",
		"gateway timeout",
		"network",
		"dial tcp",
		"eof",
	) {
		return channelKeyFailureRetry
	}

	return channelKeyFailureRetry
}

func containsAny(message string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(message, keyword) {
			return true
		}
	}
	return false
}

func truncateChannelKeyError(errMessage string) string {
	const maxLength = 1024
	if len(errMessage) <= maxLength {
		return errMessage
	}
	return errMessage[:maxLength]
}

func nextChannelKeyCursor(channelId int, total int) int {
	if total <= 1 {
		return 0
	}
	if common.RedisEnabled {
		cursor, err := common.RedisIncrease(fmt.Sprintf("channel_key_cursor:%d", channelId))
		if err == nil {
			return int((cursor - 1) % int64(total))
		}
		logger.SysError("failed to advance redis channel key cursor: " + err.Error())
	}
	counter, _ := channelKeyLocalCursor.LoadOrStore(channelId, &atomic.Uint64{})
	value := counter.(*atomic.Uint64).Add(1)
	return int((value - 1) % uint64(total))
}
