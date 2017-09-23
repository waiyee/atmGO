package main

import (
	"time"
	"atm/bittrex"
	"atm/db"
	"gopkg.in/mgo.v2/bson"
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
				session := mydb.Session.Clone()
				defer session.Close()
				e := session.DB("v2").C("ErrorLog").With(session)
				e.Insert(&db.ErrorLog{Description:"Get summaries - API", Error:err.Error(), Time:time.Now()})
			}

			session := mydb.Session.Clone()
			defer session.Close()
			d := session.DB("v2").C("OwnOrderBook2").With(session)

			thisSM.Lock.Lock()
			defer thisSM.Lock.Unlock()
			for _,v:= range markets {
				thisSM.Markets[v.MarketName] = v

				err := d.Update(bson.M{"marketname":v.MarketName}, bson.M{"$set" : bson.M{"currentrate": v.Last, "updatedat": time.Now()}})
				if err != nil && err.Error()!= "not found" {
					error := session.DB("v2").C("ErrorLog").With(session)
					error.Insert(&db.ErrorLog{Description:"Update current price", Error:err.Error(), Time:time.Now()})
				}

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
