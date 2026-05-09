package model

import (
	"strings"

	"github.com/songquanpeng/one-api/common/utils"
)

func normalizeModelIDPrefixV2(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	prefix = strings.Trim(prefix, "-")
	return prefix
}

func splitChannelModelsV2(models string) []string {
	rawModels := strings.Split(models, ",")
	normalizedModels := make([]string, 0, len(rawModels))
	for _, modelName := range rawModels {
		modelName = strings.TrimSpace(modelName)
		if modelName == "" {
			continue
		}
		normalizedModels = append(normalizedModels, modelName)
	}
	return utils.DeDuplication(normalizedModels)
}

func stripModelIDPrefixV2(modelName string, prefix string) string {
	prefix = normalizeModelIDPrefixV2(prefix)
	if prefix == "" {
		return modelName
	}
	prefixWithSeparator := prefix + "-"
	if strings.HasPrefix(modelName, prefixWithSeparator) {
		return strings.TrimPrefix(modelName, prefixWithSeparator)
	}
	return modelName
}

func applyModelIDPrefixV2(modelName string, prefix string) string {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return ""
	}
	prefix = normalizeModelIDPrefixV2(prefix)
	if prefix == "" {
		return modelName
	}
	prefixWithSeparator := prefix + "-"
	if strings.HasPrefix(modelName, prefixWithSeparator) {
		return modelName
	}
	return prefixWithSeparator + modelName
}

func recoverRawModelNamesV2(modelNames []string, previousPrefix string) []string {
	rawModels := make([]string, 0, len(modelNames))
	for _, modelName := range modelNames {
		rawModelName := stripModelIDPrefixV2(modelName, previousPrefix)
		if rawModelName == "" {
			continue
		}
		rawModels = append(rawModels, rawModelName)
	}
	return utils.DeDuplication(rawModels)
}

func applyModelIDPrefixToModelsV2(modelNames []string, prefix string) []string {
	prefixedModels := make([]string, 0, len(modelNames))
	for _, modelName := range modelNames {
		prefixedModelName := applyModelIDPrefixV2(modelName, prefix)
		if prefixedModelName == "" {
			continue
		}
		prefixedModels = append(prefixedModels, prefixedModelName)
	}
	return utils.DeDuplication(prefixedModels)
}

func PrepareChannelModelIDPrefixForWriteV2(channel *Channel, previousChannel *Channel) {
	if channel == nil {
		return
	}

	previousPrefix := ""
	if previousChannel != nil {
		previousPrefix = previousChannel.GetModelIDPrefix()
	}

	targetPrefix := previousPrefix
	prefixExplicitlyProvided := channel.ModelIDPrefix != nil
	if channel.ModelIDPrefix != nil {
		targetPrefix = normalizeModelIDPrefixV2(*channel.ModelIDPrefix)
	}
	switch {
	case prefixExplicitlyProvided:
		channel.ModelIDPrefix = &targetPrefix
	case targetPrefix == "":
		channel.ModelIDPrefix = nil
	default:
		channel.ModelIDPrefix = &targetPrefix
	}

	modelsCSV := channel.Models
	if strings.TrimSpace(modelsCSV) == "" && previousChannel != nil {
		modelsCSV = previousChannel.Models
	}

	modelNames := splitChannelModelsV2(modelsCSV)
	rawModelNames := recoverRawModelNamesV2(modelNames, previousPrefix)
	prefixedModelNames := applyModelIDPrefixToModelsV2(rawModelNames, targetPrefix)
	channel.Models = strings.Join(prefixedModelNames, ",")
}

func buildAutoModelMappingV2(modelNames []string, prefix string) map[string]string {
	prefix = normalizeModelIDPrefixV2(prefix)
	if prefix == "" {
		return nil
	}
	autoMapping := make(map[string]string)
	for _, modelName := range modelNames {
		rawModelName := stripModelIDPrefixV2(modelName, prefix)
		if rawModelName == "" || rawModelName == modelName {
			continue
		}
		autoMapping[modelName] = rawModelName
	}
	if len(autoMapping) == 0 {
		return nil
	}
	return autoMapping
}

func mergeModelMappingsV2(autoMapping map[string]string, manualMapping map[string]string) map[string]string {
	if len(autoMapping) == 0 && len(manualMapping) == 0 {
		return nil
	}
	mergedMapping := make(map[string]string, len(autoMapping)+len(manualMapping))
	for key, value := range autoMapping {
		mergedMapping[key] = value
	}
	for key, value := range manualMapping {
		mergedMapping[key] = value
	}
	return mergedMapping
}
