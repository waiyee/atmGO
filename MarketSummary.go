package main

import (
	"fmt"
	"time"
	"atm/bittrex"
)

func CompareMarkets(){
	MMPB.Lock.Lock()
	defer MMPB.Lock.Unlock()
	for _,v := range thisSM.Markets{

		if v.BaseVolume != lastSM.Markets[v.MarketName].BaseVolume || v.Volume != lastSM.Markets[v.MarketName].Volume {
			//fmt.Println("bv:", v.BaseVolume, "last bv:", lastSecondMarket[v.MarketName].BaseVolume, "v:", v.Volume, "last v", lastSecondMarket[v.MarketName].Volume)
			ATP := (v.BaseVolume-lastSM.Markets[v.MarketName].BaseVolume) / (v.Volume-lastSM.Markets[v.MarketName].Volume)
			MP := (v.Bid + v.Ask) / 2
			LMP := (lastSM.Markets[v.MarketName].Bid + lastSM.Markets[v.MarketName].Ask) / 2
			AMP := (MP + LMP) / 2
			MPB := ATP - AMP
			MMPB.Markets[v.MarketName] = MPB
		}

	}
	lastSM.Lock.Lock()
	defer lastSM.Lock.Unlock()
	for k, v := range thisSM.Markets {
		lastSM.Markets[k] = v
	}
}


func loopGetSummary() {
	/** Keep calling periodicGetSummaries every second in async mode **/
	for t := range time.NewTicker(time.Second ).C {
		go func() {
			b := bittrex.New(API_KEY, API_SECRET)

			markets, err := b.GetMarketSummaries()
			if err != nil {
				fmt.Println("periodicGetSummaries", time.Now(), err)
			}

			thisSM.Lock.Lock()
			defer thisSM.Lock.Unlock()
			for _,v:= range markets {
				thisSM.Markets[v.MarketName] = v
			}


			if len(lastSM.Markets) == 0 {
				lastSM.Lock.Lock()
				defer lastSM.Lock.Unlock()
				for k, v := range thisSM.Markets {
					lastSM.Markets[k] = v
				}

			}else{

				CompareMarkets()
			}

		}()

		JobChannel <- t
	}
}
