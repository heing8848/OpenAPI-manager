package model

import "fmt"

// EnableChannelKeyV2 allows admins to manually recover a disabled key without
// deleting and recreating it. We reuse the existing V2 success-state write path
// so cache, mirror key, and key health state stay consistent.
func EnableChannelKeyV2(channelId int, channelKeyId int) (*ChannelKey, error) {
	if channelId == 0 {
		return nil, fmt.Errorf("channel id is required")
	}
	if channelKeyId == 0 {
		return nil, fmt.Errorf("channel key id is required")
	}

	if err := MarkChannelKeySuccessV2(channelId, channelKeyId); err != nil {
		return nil, err
	}

	var key ChannelKey
	if err := DB.Where("channel_id = ? AND id = ?", channelId, channelKeyId).First(&key).Error; err != nil {
		return nil, err
	}
	return &key, nil
}
