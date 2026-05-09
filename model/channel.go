package model

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"gorm.io/gorm"
)

const (
	ChannelStatusUnknown          = 0
	ChannelStatusEnabled          = 1 // don't use 0, 0 is the default value!
	ChannelStatusManuallyDisabled = 2 // also don't use 0
	ChannelStatusAutoDisabled     = 3
)

type Channel struct {
	Id                 int          `json:"id"`
	Type               int          `json:"type" gorm:"default:0"`
	Key                string       `json:"key" gorm:"type:text"`
	Status             int          `json:"status" gorm:"default:1"`
	Name               string       `json:"name" gorm:"index"`
	Weight             *uint        `json:"weight" gorm:"default:0"`
	CreatedTime        int64        `json:"created_time" gorm:"bigint"`
	TestTime           int64        `json:"test_time" gorm:"bigint"`
	ResponseTime       int          `json:"response_time"` // in milliseconds
	BaseURL            *string      `json:"base_url" gorm:"column:base_url;default:''"`
	Other              *string      `json:"other"`   // DEPRECATED: please save config to field Config
	Balance            float64      `json:"balance"` // in USD
	BalanceUpdatedTime int64        `json:"balance_updated_time" gorm:"bigint"`
	Models             string       `json:"models"`
	ModelIDPrefix      *string      `json:"model_id_prefix" gorm:"column:model_id_prefix;type:varchar(128);default:''"`
	DisplayOrder       *int64       `json:"display_order" gorm:"column:display_order;bigint;default:0"`
	Group              string       `json:"group" gorm:"type:varchar(32);default:'default'"`
	UsedQuota          int64        `json:"used_quota" gorm:"bigint;default:0"`
	ModelMapping       *string      `json:"model_mapping" gorm:"type:varchar(1024);default:''"`
	Priority           *int64       `json:"priority" gorm:"bigint;default:0"`
	Config             string       `json:"config"`
	SystemPrompt       *string      `json:"system_prompt" gorm:"type:text"`
	Failures           int          `json:"failures" gorm:"-"`
	HasDisabledKeys    bool         `json:"has_disabled_keys" gorm:"-"`
	Keys               []ChannelKey `json:"keys,omitempty" gorm:"foreignKey:ChannelId;constraint:OnDelete:CASCADE"`
}

type ChannelConfig struct {
	Region            string `json:"region,omitempty"`
	SK                string `json:"sk,omitempty"`
	AK                string `json:"ak,omitempty"`
	UserID            string `json:"user_id,omitempty"`
	UpstreamProxy     string `json:"upstream_proxy,omitempty"`
	APIVersion        string `json:"api_version,omitempty"`
	LibraryID         string `json:"library_id,omitempty"`
	Plugin            string `json:"plugin,omitempty"`
	VertexAIProjectID string `json:"vertex_ai_project_id,omitempty"`
	VertexAIADC       string `json:"vertex_ai_adc,omitempty"`
	EdgeProxy         bool   `json:"edge_proxy,omitempty"`
}

func GetAllChannels(startIdx int, num int, scope string) ([]*Channel, error) {
	var channels []*Channel
	var err error
	switch scope {
	case "all":
		err = DB.Order("id desc").Omit("key").Find(&channels).Error
	case "disabled":
		err = DB.Order("id desc").Where("status = ? or status = ?", ChannelStatusAutoDisabled, ChannelStatusManuallyDisabled).Omit("key").Find(&channels).Error
	default:
		err = DB.Order("id desc").Limit(num).Offset(startIdx).Omit("key").Find(&channels).Error
	}
	return channels, err
}

func SearchChannels(keyword string) (channels []*Channel, err error) {
	err = DB.Omit("key").Where("id = ? or name LIKE ?", helper.String2Int(keyword), keyword+"%").Find(&channels).Error
	return channels, err
}

func GetChannelById(id int, selectAll bool) (*Channel, error) {
	channel := Channel{Id: id}
	var err error = nil
	if selectAll {
		err = DB.Preload("Keys", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("position asc, id asc")
		}).First(&channel, "id = ?", id).Error
	} else {
		err = DB.Omit("key").First(&channel, "id = ?", id).Error
	}
	return &channel, err
}

func BatchInsertChannels(channels []Channel) error {
	for i := range channels {
		if err := channels[i].Insert(); err != nil {
			return err
		}
	}
	return nil
}

func (channel *Channel) GetPriority() int64 {
	if channel.Priority == nil {
		return 0
	}
	return *channel.Priority
}

func (channel *Channel) GetBaseURL() string {
	if channel.BaseURL == nil {
		return ""
	}
	return *channel.BaseURL
}

func (channel *Channel) GetModelIDPrefix() string {
	if channel.ModelIDPrefix == nil {
		return ""
	}
	return normalizeModelIDPrefixV2(*channel.ModelIDPrefix)
}

func (channel *Channel) GetModelMapping() map[string]string {
	if channel.ModelMapping == nil || *channel.ModelMapping == "" || *channel.ModelMapping == "{}" {
		return nil
	}
	modelMapping := make(map[string]string)
	err := json.Unmarshal([]byte(*channel.ModelMapping), &modelMapping)
	if err != nil {
		logger.SysError(fmt.Sprintf("failed to unmarshal model mapping for channel %d, error: %s", channel.Id, err.Error()))
		return nil
	}
	return modelMapping
}

func (channel *Channel) GetModelMappingV2() map[string]string {
	manualMapping := channel.GetModelMapping()
	autoMapping := buildAutoModelMappingV2(splitChannelModelsV2(channel.Models), channel.GetModelIDPrefix())
	return mergeModelMappingsV2(autoMapping, manualMapping)
}

func (channel *Channel) Insert() error {
	keyValues := SplitLegacyChannelKeys(channel.Key)
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := prepareChannelDisplayOrderForInsertV2(tx, channel); err != nil {
			return err
		}
		if err := tx.Omit("Keys").Create(channel).Error; err != nil {
			return err
		}
		if err := upsertChannelKeysTx(tx, channel, keyValues); err != nil {
			return err
		}
		return channel.addAbilitiesTx(tx)
	})
}

func (channel *Channel) Update() error {
	keyValues := SplitLegacyChannelKeys(channel.Key)
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&Channel{Id: channel.Id}).Omit("Keys").Updates(channel).Error; err != nil {
			return err
		}
		if err := upsertChannelKeysTx(tx, channel, keyValues); err != nil {
			return err
		}
		if err := tx.Preload("Keys", func(db *gorm.DB) *gorm.DB {
			return db.Order("position asc, id asc")
		}).First(channel, "id = ?", channel.Id).Error; err != nil {
			return err
		}
		return channel.updateAbilitiesTx(tx)
	})
}

func (channel *Channel) UpdateResponseTime(responseTime int64) {
	err := DB.Model(channel).Select("response_time", "test_time").Updates(Channel{
		TestTime:     helper.GetTimestamp(),
		ResponseTime: int(responseTime),
	}).Error
	if err != nil {
		logger.SysError("failed to update response time: " + err.Error())
	}
}

func (channel *Channel) UpdateBalance(balance float64) {
	err := DB.Model(channel).Select("balance_updated_time", "balance").Updates(Channel{
		BalanceUpdatedTime: helper.GetTimestamp(),
		Balance:            balance,
	}).Error
	if err != nil {
		logger.SysError("failed to update balance: " + err.Error())
	}
}

func (channel *Channel) Delete() error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("channel_id = ?", channel.Id).Delete(&ChannelKey{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(channel).Error; err != nil {
			return err
		}
		return channel.deleteAbilitiesTx(tx)
	})
}

func (channel *Channel) LoadConfig() (ChannelConfig, error) {
	var cfg ChannelConfig
	if channel.Config == "" {
		return cfg, nil
	}
	err := json.Unmarshal([]byte(channel.Config), &cfg)
	if err != nil {
		return cfg, err
	}
	return cfg, nil
}

func UpdateChannelStatusById(id int, status int) {
	err := UpdateAbilityStatus(id, status == ChannelStatusEnabled)
	if err != nil {
		logger.SysError("failed to update ability status: " + err.Error())
	}
	err = DB.Model(&Channel{}).Where("id = ?", id).Update("status", status).Error
	if err != nil {
		logger.SysError("failed to update channel status: " + err.Error())
	}
}

func UpdateChannelUsedQuota(id int, quota int64) {
	if config.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeChannelUsedQuota, id, quota)
		return
	}
	updateChannelUsedQuota(id, quota)
}

func updateChannelUsedQuota(id int, quota int64) {
	err := DB.Model(&Channel{}).Where("id = ?", id).Update("used_quota", gorm.Expr("used_quota + ?", quota)).Error
	if err != nil {
		logger.SysError("failed to update channel used quota: " + err.Error())
	}
}

func DeleteChannelByStatus(status int64) (int64, error) {
	var rowsAffected int64
	err := DB.Transaction(func(tx *gorm.DB) error {
		subQuery := tx.Model(&Channel{}).Select("id").Where("status = ?", status)
		if err := tx.Where("channel_id IN (?)", subQuery).Delete(&ChannelKey{}).Error; err != nil {
			return err
		}
		result := tx.Where("status = ?", status).Delete(&Channel{})
		rowsAffected = result.RowsAffected
		return result.Error
	})
	return rowsAffected, err
}

func DeleteDisabledChannel() (int64, error) {
	var rowsAffected int64
	err := DB.Transaction(func(tx *gorm.DB) error {
		subQuery := tx.Model(&Channel{}).Select("id").Where("status = ? or status = ?", ChannelStatusAutoDisabled, ChannelStatusManuallyDisabled)
		if err := tx.Where("channel_id IN (?)", subQuery).Delete(&ChannelKey{}).Error; err != nil {
			return err
		}
		result := tx.Where("status = ? or status = ?", ChannelStatusAutoDisabled, ChannelStatusManuallyDisabled).Delete(&Channel{})
		rowsAffected = result.RowsAffected
		return result.Error
	})
	return rowsAffected, err
}

func (channel *Channel) HasMultipleLegacyKeys() bool {
	return strings.Contains(channel.Key, "\n") || strings.Contains(channel.Key, "\r")
}
