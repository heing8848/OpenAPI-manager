package monitor

import (
	"sync"
	"time"
)

var (
	channelFailures     = make(map[int][]time.Time)
	channelFailuresLock = sync.RWMutex{}
)

func RecordAndGetChannelFailures(channelId int) int {
	channelFailuresLock.Lock()
	defer channelFailuresLock.Unlock()

	now := time.Now()
	var recent []time.Time

	if timestamps, ok := channelFailures[channelId]; ok {
		for _, t := range timestamps {
			if now.Sub(t) <= time.Hour {
				recent = append(recent, t)
			}
		}
	}

	recent = append(recent, now)
	channelFailures[channelId] = recent
	return len(recent)
}

func ClearChannelFailures(channelId int) {
	channelFailuresLock.Lock()
	defer channelFailuresLock.Unlock()
	delete(channelFailures, channelId)
}

func GetChannelFailures(channelId int) int {
	channelFailuresLock.RLock()
	defer channelFailuresLock.RUnlock()

	now := time.Now()
	var count int

	if timestamps, ok := channelFailures[channelId]; ok {
		for _, t := range timestamps {
			if now.Sub(t) <= time.Hour {
				count++
			}
		}
	}

	return count
}
