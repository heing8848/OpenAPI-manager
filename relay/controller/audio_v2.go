package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/relay/meta"
	relaymodel "github.com/songquanpeng/one-api/relay/model"
)

func RelayAudioHelperV2(c *gin.Context, relayMode int) *relaymodel.ErrorWithStatusCode {
	metaInfo := meta.GetByContext(c)
	if capabilityErr := ValidateStandaloneOutputCapabilityV2(metaInfo, relayMode); capabilityErr != nil {
		return capabilityErr
	}
	logger.Infof(
		c.Request.Context(),
		"processing standalone audio request via channel_type=%s channel_id=%d model=%s mode=%d",
		getStandaloneOutputChannelNameV2(metaInfo.ChannelType),
		metaInfo.ChannelId,
		getStandaloneOutputModelNameV2(metaInfo),
		relayMode,
	)
	return RelayAudioHelper(c, relayMode)
}
