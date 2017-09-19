package main

import (
	"time"
	"fmt"
	"sync"
	"atm/bittrex"

	"atm/db"

	"gopkg.in/mgo.v2/bson"
)

var minTotal = float64(0.00060000)

/**
* Loop the order book limit to 8 markets per second
 */
func loopGetOrderBook()  {
	lenOfM := len(BTCMarkets)
	start := 0
	end := -1
	limit := 8
	for t := range time.NewTicker(time.Second).C {
		var OrderMarkets []string
		if end + 1 < lenOfM && end + limit< lenOfM {
			start = end + 1
			end += limit
			OrderMarkets = BTCMarkets[start:end+1]
		} else if end + 1 < lenOfM {
			start = end + 1
			end = lenOfM -1
			OrderMarkets = BTCMarkets[start:end+1]
			end = limit - ( end - start +1 ) -1
			start = 0
			for i:= start ; i <= end ; i ++ {
				OrderMarkets = append(OrderMarkets, BTCMarkets[i])
			}

		} else {
			start = 0
			end = -1 + limit
			OrderMarkets = BTCMarkets[start:end+1]
		}


		go periodicGetOrderBook(t, OrderMarkets)
		JobChannel <- t

	}

}


type WalletBalance struct {
	Available      float64    `json:"available" bson:"available"`
}


func periodicGetOrderBook(t time.Time, markets []string)  {
	wg := &sync.WaitGroup{}

	obRate := 0.125

	bittrex := bittrex.New(API_KEY, API_SECRET)

	for i := 0; i < len(markets); i++ {
		wg.Add(1)
		go func(wg *sync.WaitGroup, i int) {

			orderBook, err := bittrex.GetOrderBook(markets[i], "both")
			if err != nil {
				fmt.Println("periodicGetOrderBook - ", time.Now(), err)
			}else if len(orderBook.Buy) > 0 && len(orderBook.Sell) > 0 {
				bidVol := float64(0)

				midPrice := ( orderBook.Buy[0].Rate + orderBook.Sell[0].Rate) / 2.0

				quBidRate := midPrice * (1 - obRate)
				quAskRate := midPrice * (1 + obRate)

				for v := 0; v < len(orderBook.Buy); v++ {
					if orderBook.Buy[v].Rate >= quBidRate {
						bidVol += orderBook.Buy[v].Quantity * orderBook.Buy[v].Rate
					} else {
						break
					}
				}

				askVol := float64(0)

				for v := 0; v < len(orderBook.Sell); v++ {
					if orderBook.Sell[v].Rate <= quAskRate {
						askVol += orderBook.Sell[v].Quantity * orderBook.Sell[v].Rate
					} else {
						break
					}
				}
				VB := float64(0)
				VA := float64(0)
				thisSM.Lock.Lock()
				lastSM.Lock.Lock()
				Pbt := thisSM.Markets[markets[i]].Bid
				Pbt1 := lastSM.Markets[markets[i]].Bid
				Pat := thisSM.Markets[markets[i]].Ask
				Pat1 := lastSM.Markets[markets[i]].Ask
				thisSM.Lock.Unlock()
				lastSM.Lock.Unlock()
				if Pbt < Pbt1{
					VB = 0
				}else if Pbt > Pbt1{
					VB = bidVol
				}else{
					VB = 0
				}

				if Pat < Pat1{
					VA = askVol
				}else if Pat > Pat1{
					VA = 0
				}else{
					VA = 0
				}

				VOI := float64(0)
				VOI = VB - VA
				OIR := float64(0)
				OIR = (bidVol - askVol) / (bidVol + askVol)
				Spread := orderBook.Sell[0].Rate - orderBook.Buy[0].Rate
				Spread = Spread * 100000000
				MMPB.Lock.Lock()
				defer MMPB.Lock.Unlock()
				MPB := MMPB.Markets[markets[i]]

				final := (VOI / Spread) + (OIR / Spread ) + (MPB / Spread)

				session := mydb.Session.Clone()
				defer session.Close()
				c := session.DB("v2").C("OwnOrderBook").With(session)

				/** final may need to adjust to obtain a better result
					e.g. (final / 10000000) > 0.2
					need to test
				*/
				boughtOrder := []db.Orders{}
				err = c.Find(bson.M{
					"marketname" : markets[i],
					"status" : "bought",

				}).All(&boughtOrder)


				if err != nil{
					fmt.Println("Select buying selling market ", time.Now(),err)
				}
				var BTCBalance WalletBalance
				d := session.DB("v2").C("WalletBalance").With(session)
				err = d.Find(bson.M{
					"currency" : "BTC",
				}).Select(bson.M{"available":1}).One(&BTCBalance)
				if err != nil{
					fmt.Println("Select buying selling market ", time.Now(),err)
				}

				if final > 0 && len(boughtOrder) == 0 && BTCBalance.Available >= minTotal{
					fmt.Printf("Bought Market: %v , VOI: %f, OIR: %f, MPB: %f, Spread: %f, Final : %f \n", markets[i],VOI,OIR,MPB,Spread,final)
					// place buy order at ask rate
					rate := orderBook.Sell[0].Rate
					// Attention for not enough balance?
					quantity := (minTotal * (1-fee)) / rate
					uuid := "xyz" // get from bittrex api
					ofee := rate * quantity * fee
					total := ( rate * quantity ) + ofee
					wallet := BTCBalance.Available - total
					buyorder := &db.Orders{
						Status : "buying",
						MarketName : markets[i],
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
						Buy: db.OrderBook{
							UUID: uuid,
							Status: "bought",
							Quantity: quantity,
							Rate: rate,
							Fee: ofee,
							Total: total,
							OrderTime: time.Now(),
						},
					}
					err := c.Insert(&buyorder)
					err2 := d.Update(bson.M{"currency":"BTC"}, bson.M{"$set" : bson.M{"available": wallet}})
					if err != nil {
						fmt.Println("Place buy order - ", time.Now(), err)
					}
					if err2!= nil{
						fmt.Println("Update Wallet - ", time.Now(), err2)
					}
				}else if final < 0 && len(boughtOrder) > 0 {
					fmt.Printf("Sold Market: %v , VOI: %f, OIR: %f, MPB: %f, Spread: %f, Final : %f \n", markets[i],VOI,OIR,MPB,Spread,final)
					// if stocks on hand
					// place sell order at bid rate
					sellOrder := &boughtOrder[0]
					rate := orderBook.Buy[0].Rate
					quantity := sellOrder.Buy.Quantity
					ofee := rate * quantity
					total := (rate*quantity) - ofee
					wallet := BTCBalance.Available + total
					sellOrder.Sell.Status = "sold"
					sellOrder.Status = "sold"
					sellOrder.Sell.Rate = rate
					sellOrder.Sell.Quantity = quantity
					sellOrder.Sell.Fee = ofee
					sellOrder.Sell.Total = total

					err := c.Update(bson.M{ "_id" : boughtOrder[0].Id}, &sellOrder)
					err2 := d.Update(bson.M{"currency":"BTC"}, bson.M{"$set" : bson.M{"available": wallet}})
					if err != nil {
						fmt.Println("Place buy order - ", time.Now(), err)
					}
					if err2!= nil{
						fmt.Println("Update Wallet - ", time.Now(), err2)
					}
				}







			}
			//defer wgm.Done()
			wg.Done()
		}(wg, i)

	}

	wg.Wait()


}



func refreshOrder(){
	for t:= range time.NewTicker(time.Millisecond * 125 ).C{
		// Prepare to update order status from bittrex
		bapi := bittrex.New(API_KEY, API_SECRET)
		_, err := bapi.GetOrder("dc7db5c5-37b7-4fbf-b619-15a2b0b23dbe")

		if err != nil{
			fmt.Println("Refresh order ", time.Now(), err)
		}

		JobChannel <- t
	}
}