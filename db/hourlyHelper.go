package db

import (
	"sync"
	"time"
)

type RateWithLock struct{
	Lock 	sync.Mutex
	HMR		HourMarketRate
}

type RateWithMarketName struct{
	MarketName string
	HMR 	HourMarketRate
}

type HourMarketRate struct{

	MinAsk 	float64
	MaxAsk 	float64
	LastAsk	float64
	MinBid	float64
	MaxBid	float64
	LastBid	float64
	MinFinal	float64
	MaxFinal	float64
	LastFinal	float64
	LogDetail 	[]hourlyLog
}

type hourlyLog struct {
	LogTime	time.Time
	Bid 	float64
	Ask		float64
	Final	float64
}

func (h *HourMarketRate) New(){
	h.MaxAsk = -99999
	h.MaxBid = -99999
	h.MaxFinal = -99999
	h.MinAsk = 99999
	h.MinBid = 99999
	h.MinFinal = 99999
}


func (h *HourMarketRate) InsertLog(b float64, a float64, f float64){
	h.LogDetail = append(h.LogDetail, hourlyLog{LogTime:time.Now(), Bid:b, Ask:a, Final:f})
	h.LastAsk = a
	h.LastBid = b
	h.LastFinal = f
	for i, v := range h.LogDetail{
		if time.Now().Sub(v.LogTime).Minutes() > 180{
			h.LogDetail = append(h.LogDetail[:i], h.LogDetail[i+1:]...)
		}
	}
	for _, v := range h.LogDetail{
		if v.Ask < h.MinAsk {
			h.MinAsk = v.Ask
		}
		if v.Bid < h.MinBid{
			h.MinBid = v.Bid
		}
		if v.Final < h.MinFinal{
			h.MinFinal = v.Final
		}
		if v.Ask > h.MaxAsk{
			h.MaxAsk = v.Ask
		}
		if v.Bid > h.MaxBid{
			h.MaxBid = v.Bid
		}
		if v.Final > h.MaxFinal{
			h.MaxFinal = v.Final
		}

	}

}