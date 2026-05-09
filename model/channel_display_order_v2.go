package model

import (
	"errors"

	"github.com/songquanpeng/one-api/common/helper"
	"gorm.io/gorm"
)

func (channel *Channel) GetDisplayOrderV2() int64 {
	if channel == nil || channel.DisplayOrder == nil {
		return 0
	}
	return *channel.DisplayOrder
}

func GetAllChannelsV2(startIdx int, num int, scope string) ([]*Channel, error) {
	if err := EnsureChannelDisplayOrdersV2(); err != nil {
		return nil, err
	}
	var channels []*Channel
	query := DB.Order("display_order asc").Order("id asc").Omit("key")
	switch scope {
	case "all":
		return channels, query.Find(&channels).Error
	case "disabled":
		return channels, query.Where("status = ? or status = ?", ChannelStatusAutoDisabled, ChannelStatusManuallyDisabled).Find(&channels).Error
	default:
		return channels, query.Limit(num).Offset(startIdx).Find(&channels).Error
	}
}

func SearchChannelsV2(keyword string) ([]*Channel, error) {
	if err := EnsureChannelDisplayOrdersV2(); err != nil {
		return nil, err
	}
	var channels []*Channel
	err := DB.Order("display_order asc").Order("id asc").Omit("key").Where("id = ? or name LIKE ?", helper.String2Int(keyword), keyword+"%").Find(&channels).Error
	return channels, err
}

func EnsureChannelDisplayOrdersV2() error {
	return DB.Transaction(func(tx *gorm.DB) error {
		return ensureChannelDisplayOrdersTxV2(tx)
	})
}

func MoveChannelDisplayOrderV2(channelId int, direction int) error {
	if direction != -1 && direction != 1 {
		return errors.New("invalid direction")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := ensureChannelDisplayOrdersTxV2(tx); err != nil {
			return err
		}

		var channels []Channel
		if err := tx.Order("display_order asc").Order("id asc").Find(&channels).Error; err != nil {
			return err
		}

		currentIndex := -1
		for index := range channels {
			if channels[index].Id == channelId {
				currentIndex = index
				break
			}
		}
		if currentIndex == -1 {
			return gorm.ErrRecordNotFound
		}

		targetIndex := currentIndex + direction
		if targetIndex < 0 || targetIndex >= len(channels) {
			return nil
		}

		currentOrder := channels[currentIndex].GetDisplayOrderV2()
		targetOrder := channels[targetIndex].GetDisplayOrderV2()
		if err := tx.Model(&Channel{}).Where("id = ?", channels[currentIndex].Id).Update("display_order", targetOrder).Error; err != nil {
			return err
		}
		return tx.Model(&Channel{}).Where("id = ?", channels[targetIndex].Id).Update("display_order", currentOrder).Error
	})
}

func prepareChannelDisplayOrderForInsertV2(tx *gorm.DB, channel *Channel) error {
	if channel == nil {
		return nil
	}
	if channel.GetDisplayOrderV2() > 0 {
		return nil
	}
	if err := ensureChannelDisplayOrdersTxV2(tx); err != nil {
		return err
	}
	var maxDisplayOrder int64
	if err := tx.Model(&Channel{}).Select("COALESCE(MAX(display_order), 0)").Scan(&maxDisplayOrder).Error; err != nil {
		return err
	}
	nextDisplayOrder := maxDisplayOrder + 1
	channel.DisplayOrder = &nextDisplayOrder
	return nil
}

func ensureChannelDisplayOrdersTxV2(tx *gorm.DB) error {
	var channels []Channel
	if err := tx.Order("CASE WHEN COALESCE(display_order, 0) > 0 THEN 0 ELSE 1 END asc").Order("display_order asc").Order("id desc").Find(&channels).Error; err != nil {
		return err
	}
	if len(channels) == 0 {
		return nil
	}

	seen := make(map[int64]bool)
	needsRebuild := false
	for _, channel := range channels {
		displayOrder := channel.GetDisplayOrderV2()
		if displayOrder <= 0 || seen[displayOrder] {
			needsRebuild = true
			break
		}
		seen[displayOrder] = true
	}
	if !needsRebuild {
		return nil
	}

	for index := range channels {
		displayOrder := int64(index + 1)
		if err := tx.Model(&Channel{}).Where("id = ?", channels[index].Id).Update("display_order", displayOrder).Error; err != nil {
			return err
		}
	}
	return nil
}
