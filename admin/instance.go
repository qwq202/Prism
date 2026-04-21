package admin

import "chat/globals"

var MarketInstance *Market

func InitInstance() {
	MarketInstance = NewMarket()
	globals.VisionModelResolver = func(model string) bool {
		if MarketInstance == nil {
			return false
		}

		item := MarketInstance.GetModel(model)
		return item != nil && item.VisionModel
	}
}
