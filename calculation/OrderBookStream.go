package calculation

import "time"


type OrderBookStream struct {
	GetTime time.Time
	MarketName string
	MidPrice float64
	BidRate float64
	AskRate float64
	BidVol float64
	AskVol float64
}
