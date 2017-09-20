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
			session := mydb.Session.Clone()
			defer session.Close()

			orderBook, err := bittrex.GetOrderBook(markets[i], "both")
			if err != nil {

				e := session.DB("v2").C("ErrorLog").With(session)
				e.Insert(&db.ErrorLog{Description:"PeriodicGetOrderbook - API ", Error:err.Error(), Time:time.Now()})

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

					e := session.DB("v2").C("ErrorLog").With(session)
					e.Insert(&db.ErrorLog{Description:"Periodic get order Select bought orders", Error:err.Error(), Time:time.Now()})

				}
				var BTCBalance WalletBalance
				d := session.DB("v2").C("WalletBalance").With(session)
				err = d.Find(bson.M{
					"currency" : "BTC",
				}).Select(bson.M{"available":1}).One(&BTCBalance)
				if err != nil{

					e := session.DB("v2").C("ErrorLog").With(session)
					e.Insert(&db.ErrorLog{Description:"Get BTC Balance of DB", Error:err.Error(), Time:time.Now()})
				}

				threshold := float64(0)
				threshold = 0.2
				minRate := float64(0.000001)

				if final > threshold && len(boughtOrder) == 0 && BTCBalance.Available >= minTotal && orderBook.Sell[0].Rate > minRate{
					fmt.Printf("Bought Market: %v , VOI: %f, OIR: %f, MPB: %f, Spread: %f, Final : %f \n", markets[i],VOI,OIR,MPB,Spread,final)
					// place buy order at ask rate
					rate := orderBook.Sell[0].Rate
					// Attention for not enough balance?
					quantity := (minTotal * (1-fee)) / rate
					//uuid := "xyz" // get from bittrex api
					uuid,placeOErr := bittrex.BuyLimit(markets[i],quantity, rate)
					if placeOErr!= err{
					e := session.DB("v2").C("ErrorLog").With(session)
					e.Insert(&db.ErrorLog{Description:"Place Buy Limit - API", Error:placeOErr.Error(), Time:time.Now()})

					}
					ofee := rate * quantity * fee
					total := ( rate * quantity ) + ofee
					wallet := BTCBalance.Available - total
					buyorder := &db.Orders{
						Status : "buying",
						MarketName : markets[i],
						CurrentRate : rate,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
						Buy: db.OrderBook{
							UUID: uuid,
							Status: "buying",
							Quantity: quantity,
							Rate: rate,
							Fee: ofee,
							Total: total,
							OrderTime: time.Now(),
							Final: final,
						},
					}
					err := c.Insert(&buyorder)
					err2 := d.Update(bson.M{"currency":"BTC"}, bson.M{"$set" : bson.M{"available": wallet}})
					if err != nil {

						e := session.DB("v2").C("ErrorLog").With(session)
						e.Insert(&db.ErrorLog{Description:"Place buy order", Error:err.Error(), Time:time.Now()})

					}
					if err2!= nil{

						e := session.DB("v2").C("ErrorLog").With(session)
						e.Insert(&db.ErrorLog{Description:"Update Wallet Balace @ buy order", Error:err2.Error(), Time:time.Now()})

					}
				}else if final < -0.1 && len(boughtOrder) > 0 {
					fmt.Printf("Sold Market: %v , VOI: %f, OIR: %f, MPB: %f, Spread: %f, Final : %f \n", markets[i],VOI,OIR,MPB,Spread,final)
					// if stocks on hand
					// place sell order at bid rate
					sellOrder := &boughtOrder[0]
					rate := orderBook.Buy[0].Rate
					quantity := sellOrder.Buy.Quantity
					ofee := (rate * quantity) * fee
					total := (rate*quantity) - ofee
					wallet := BTCBalance.Available + total
					//uuid := "abc"
					uuid,placeOErr := bittrex.SellLimit(markets[i],quantity, rate)
					if placeOErr!= err{
						e := session.DB("v2").C("ErrorLog").With(session)
					e.Insert(&db.ErrorLog{Description:"Place Buy Limit - API", Error:placeOErr.Error(), Time:time.Now()})

					}
					sellOrder.Sell.UUID = uuid
					sellOrder.Sell.Status = "selling"
					sellOrder.Status = "selling"
					sellOrder.Sell.Rate = rate
					sellOrder.Sell.Quantity = quantity
					sellOrder.Sell.Fee = ofee
					sellOrder.Sell.Total = total
					sellOrder.Sell.Final = final

					err := c.Update(bson.M{ "_id" : boughtOrder[0].Id}, &sellOrder)
					err2 := d.Update(bson.M{"currency":"BTC"}, bson.M{"$set" : bson.M{"available": wallet}})
					if err != nil {

						e := session.DB("v2").C("ErrorLog").With(session)
						e.Insert(&db.ErrorLog{Description:"Place sell order", Error:err.Error(), Time:time.Now()})

					}
					if err2!= nil{

						e := session.DB("v2").C("ErrorLog").With(session)
						e.Insert(&db.ErrorLog{Description:"Update Wallet @ sell order", Error:err2.Error(), Time:time.Now()})

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
	for t:= range time.NewTicker(time.Second * 5 ).C{

		session := mydb.Session.Clone()
		defer session.Close()
		c := session.DB("v2").C("OwnOrderBook").With(session)
		sellbuyOrders := []db.Orders{}
		cerr := c.Find(bson.M{
			"status" : bson.M{"$in" : []string{"selling", "buying", "bought"}},
		}).All(&sellbuyOrders)

		if cerr != nil {

			e := session.DB("v2").C("ErrorLog").With(session)
			e.Insert(&db.ErrorLog{Description:"Get selling buying orders", Error:cerr.Error(), Time:time.Now()})

		}
		bapi := bittrex.New(API_KEY, API_SECRET)

		for _,v := range sellbuyOrders {

			var uuid string = ""

			if v.Status == "buying" {
				uuid = v.Buy.UUID
			} else if v.Status == "selling" {
				uuid = v.Sell.UUID
			}
			if uuid != "" {
				// Prepare to update order status from bittrex
				result, err := bapi.GetOrder(uuid)
				if err != nil {
					e := session.DB("v2").C("ErrorLog").With(session)
					e.Insert(&db.ErrorLog{Description: "Refresh order - API", Error: err.Error(), Time: time.Now()})
				} else {
					if !result.IsOpen {
						if v.Status == "buying" {
							v.Status = "bought"
							v.UpdatedAt = time.Now()
							v.Buy.Rate = result.PricePerUnit
							v.Buy.Fee = result.CommissionPaid
							v.Buy.Quantity = result.Quantity
							v.Buy.Total = result.Price
							v.Buy.Status = "bought"
							t, err2 := time.Parse(time.RFC3339, result.Closed)
							if err2 != nil {
								fmt.Println("Re-parse error - ", time.Now(), err2)
								v.Buy.CompleteTime = time.Now()
							} else {
								v.Buy.CompleteTime = t
							}
						} else {
							v.Status = "sold"
							v.UpdatedAt = time.Now()
							v.Sell.Rate = result.PricePerUnit
							v.Sell.Fee = result.CommissionPaid
							v.Sell.Quantity = result.Quantity
							v.Sell.Total = result.Price
							v.Sell.Status = "sold"
							t, err2 := time.Parse(time.RFC3339, result.Closed)
							if err2 != nil {
								e := session.DB("v2").C("ErrorLog").With(session)
								e.Insert(&db.ErrorLog{Description: "Re-prase time error", Error: err2.Error(), Time: time.Now()})
								v.Sell.CompleteTime = time.Now()
							} else {
								v.Sell.CompleteTime = t
							}
						}

						c.Update(bson.M{"_id": v.Id}, v)
					} else {
						// cancel buy order
						if v.Status == "buying" {
							canerr := bapi.CancelOrder(v.Buy.UUID)
							if canerr != nil {
								e := session.DB("v2").C("ErrorLog").With(session)
								e.Insert(&db.ErrorLog{Description: "Cancel order - API ", Error: canerr.Error(), Time: time.Now()})
							}
						}
					}
				}

			}else if time.Now().Sub(v.Buy.CompleteTime ).Hours() > 24 {
				price , err :=bapi.GetTicker(v.MarketName)
				if err!= nil{
					e := session.DB("v2").C("ErrorLog").With(session)
					e.Insert(&db.ErrorLog{Description: "24 hours sell order get ticker - API ", Error: err.Error(), Time: time.Now()})
				}

				rate := price.Ask
				quantity := v.Buy.Quantity
				ofee := (rate * quantity) * fee
				total := (rate*quantity) - ofee

				//uuid := "abc"
				uuid,placeOErr := bapi.SellLimit(v.MarketName,quantity, rate)
				if placeOErr!= err{
					e := session.DB("v2").C("ErrorLog").With(session)
					e.Insert(&db.ErrorLog{Description:"Place Buy Limit - API", Error:placeOErr.Error(), Time:time.Now()})

				}
				v.Sell.UUID = uuid
				v.Sell.Status = "selling"
				v.Status = "selling"
				v.Sell.Rate = rate
				v.Sell.Quantity = quantity
				v.Sell.Fee = ofee
				v.Sell.Total = total
				

				sellerr := c.Update(bson.M{ "_id" : v.Id}, &v)

				if err != nil {

					e := session.DB("v2").C("ErrorLog").With(session)
					e.Insert(&db.ErrorLog{Description:"Place sell order", Error:sellerr.Error(), Time:time.Now()})

				}


			}

		}



		JobChannel <- t
	}
}