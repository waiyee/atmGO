package main

import (
	"time"
	"fmt"
	"sync"
	"atm/bittrex"

	"atm/db"
	"atm/tradeHelper"
	"gopkg.in/mgo.v2/bson"
	"strings"
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

			for true{
				result := refreshWallet()
				if result {
					break
				}
			}

		} else {
			start = 0
			end = -1 + limit
			OrderMarkets = BTCMarkets[start:end+1]

			for true{
				result := refreshWallet()
				if result {
					break
				}
			}
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

	bapi := bittrex.New(API_KEY, API_SECRET)

	for i := 0; i < len(markets); i++ {
		wg.Add(1)
		go func(wg *sync.WaitGroup, i int) {
			session := mydb.Session.Clone()
			defer session.Close()

			orderBook, err := bapi.GetOrderBook(markets[i], "both")
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
				MPB := MMPB.Markets[markets[i]]
				MMPB.Lock.Unlock()
				final := (VOI / Spread) + (OIR / Spread ) + (MPB / Spread)

				//c := session.DB("v2").C("OwnOrderBook").With(session)


				d := session.DB("v2").C("WalletBalance").With(session)
				var MarketBalance WalletBalance
				err = d.Find(bson.M{
					"currency" : strings.Split(markets[i], "-")[1],
				}).Select(bson.M{"available":1}).One(&MarketBalance)


				if err != nil && err.Error() != "not found"{
					e := session.DB("v2").C("ErrorLog").With(session)
					e.Insert(&db.ErrorLog{Description:"Get Market Balance in DB", Error:err.Error(), Time:time.Now()})
				}else if err !=nil{
					MarketBalance.Available = 0
				}

				thisSM.Lock.Lock()
				MarketBTCEST := MarketBalance.Available * thisSM.Markets[markets[i]].Last
				thisSM.Lock.Unlock()

				var BTCBalance WalletBalance

				err = d.Find(bson.M{
					"currency" : "BTC",
				}).Select(bson.M{"available":1}).One(&BTCBalance)
				if err != nil{
					e := session.DB("v2").C("ErrorLog").With(session)
					e.Insert(&db.ErrorLog{Description:"Get BTC Balance in DB", Error:err.Error(), Time:time.Now()})
				}

				threshold := float64(0)
				threshold = 0.2
				minSellRate := float64(0.0005)
				stopLossRate := float64(0.00058)
				minRate := float64(0.00001)

				//fmt.Printf("Market %v Final %f MarketBTC %f minSellRate %f \n", markets[i], final, MarketBTCEST, minSellRate)

				if final > threshold && MarketBTCEST < minSellRate && BTCBalance.Available >= minTotal && orderBook.Sell[0].Rate > minRate{
					// buy signal
					fmt.Printf("Bought Market: %v , VOI: %f, OIR: %f, MPB: %f, Spread: %f, Final : %f \n", markets[i],VOI,OIR,MPB,Spread,final)
					// place buy order at ask rate
					rate := orderBook.Sell[0].Rate
					quantity := (minTotal * (1-fee)) / rate

					tradeHelper.BuyHelper(rate,quantity, markets[i], BTCBalance.Available, final, *bapi, mydb, "Buy Signal")

				}else if final < -0.1 && MarketBTCEST >= minSellRate {
					fmt.Printf("Sold Market: %v , VOI: %f, OIR: %f, MPB: %f, Spread: %f, Final : %f \n", markets[i],VOI,OIR,MPB,Spread,final)
					// if stocks on hand
					// place sell order at bid rate
					rate := orderBook.Buy[0].Rate
					quantity := MarketBalance.Available

					tradeHelper.SellHelper(rate,quantity, markets[i], BTCBalance.Available, final, *bapi, mydb, "Sell Signal")

				}else if MarketBTCEST >= minSellRate && MarketBTCEST < stopLossRate{
					buyingOrder := []db.Orders{}
					f := session.DB("v2").C("OwnOrderBook2").With(session)
					f.Find(bson.M{"market":markets[i], "status" :"buying"}).All(&buyingOrder)
					if len(buyingOrder) == 0 {
						fmt.Printf("Sold Market: %v , VOI: %f, OIR: %f, MPB: %f, Spread: %f, Final : %f \n", markets[i], VOI, OIR, MPB, Spread, final)
						// if stocks on hand
						// place sell order at bid rate
						rate := orderBook.Buy[0].Rate
						quantity := MarketBalance.Available

						tradeHelper.SellHelper(rate, quantity, markets[i], BTCBalance.Available, final, *bapi, mydb, "Stop Loss")
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
		c := session.DB("v2").C("OwnOrderBook2").With(session)
		sellbuyOrders := []db.Orders{}
		cerr := c.Find(bson.M{
			"status" : bson.M{"$in" : []string{"selling", "buying"}},
		}).All(&sellbuyOrders)

		if cerr != nil {
			e := session.DB("v2").C("ErrorLog").With(session)
			e.Insert(&db.ErrorLog{Description:"Get selling buying orders", Error:cerr.Error(), Time:time.Now()})

		}
		bapi := bittrex.New(API_KEY, API_SECRET)

		for _,v := range sellbuyOrders {

			var uuid string = ""

			if v.Status == "buying" {
				uuid = v.UUID
			} else if v.Status == "selling" {
				uuid = v.UUID
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
							v.Rate = result.PricePerUnit
							v.Fee = result.CommissionPaid
							v.Quantity = result.Quantity
							v.Total = result.Price
							a := strings.Split(result.Closed, ".")
							var LayoutLenMill string
							if len(a) > 1 {
								millise := len(a[1])
								LayoutLenMill = "."
								for i := 0; i < millise; i++ {
									LayoutLenMill += "0"
								}
							}
							t, err2 := time.Parse("2006-01-02T15:04:05" + LayoutLenMill, result.Closed)
							if err2 != nil {
								e := session.DB("v2").C("ErrorLog").With(session)
								e.Insert(&db.ErrorLog{Description: "Re-prase time error", Error: err2.Error(), Time: time.Now()})
								v.CompleteTime = time.Now()
							} else {
								v.CompleteTime = t
							}
						} else {
							v.Status = "sold"
							v.UpdatedAt = time.Now()
							v.Rate = result.PricePerUnit
							v.Fee = result.CommissionPaid
							v.Quantity = result.Quantity
							v.Total = result.Price
							a := strings.Split(result.Closed, ".")
							var LayoutLenMill string
							if len(a) > 1 {
								millise := len(a[1])
								LayoutLenMill = "."
								for i := 0; i < millise; i++ {
									LayoutLenMill += "0"
								}
							}
							t, err2 := time.Parse("2006-01-02T15:04:05" + LayoutLenMill, result.Closed)
							if err2 != nil {
								e := session.DB("v2").C("ErrorLog").With(session)
								e.Insert(&db.ErrorLog{Description: "Re-prase time error", Error: err2.Error(), Time: time.Now()})
								v.CompleteTime = time.Now()
							} else {
								v.CompleteTime = t
							}
						}
						err3 := c.Update(bson.M{"_id": v.Id}, v)
						if err3 != nil {
							e := session.DB("v2").C("ErrorLog").With(session)
							e.Insert(&db.ErrorLog{Description: "Update order - ", Error: err3.Error(), Time: time.Now()})
						}
					} else if result.QuantityRemaining != 0 && result.QuantityRemaining == result.Quantity{
						// cancel buy order
						if v.Status == "buying" {
							canberr := bapi.CancelOrder(v.UUID)
							if canberr != nil {
								e := session.DB("v2").C("ErrorLog").With(session)
								e.Insert(&db.ErrorLog{Description: "Cancel order - API ", Error: canberr.Error(), Time: time.Now()})
							}
						} else  if v.Status == "selling"{
							canserr := bapi.CancelOrder(v.UUID)
							if canserr != nil {
								e := session.DB("v2").C("ErrorLog").With(session)
								e.Insert(&db.ErrorLog{Description: "Cancel order - API ", Error: canserr.Error(), Time: time.Now()})
							} else {
								price , err :=bapi.GetTicker(v.MarketName)
								if err!= nil{
									e := session.DB("v2").C("ErrorLog").With(session)
									e.Insert(&db.ErrorLog{Description: "Selling too long sell order get ticker - API ", Error: err.Error(), Time: time.Now()})
								} else {

									d := session.DB("v2").C("WalletBalance").With(session)
									var BTCBalance WalletBalance

									err = d.Find(bson.M{
										"currency": "BTC",
									}).Select(bson.M{"available": 1}).One(&BTCBalance)

									rate := price.Bid
									quantity := v.Quantity

									tradeHelper.SellHelper(rate, quantity, v.MarketName, BTCBalance.Available, 0, *bapi, mydb, "Resell non-sell order")
								}
							}
						/*	ofee := (rate * quantity) * fee
							total := (rate*quantity) - ofee

							//uuid := "abc"
							uuid,placeOErr := bapi.SellLimit(v.MarketName,quantity, rate)
							if placeOErr!= err{
								e := session.DB("v2").C("ErrorLog").With(session)
								e.Insert(&db.ErrorLog{Description:"Selling too long sell Limit - API", Error:placeOErr.Error(), Time:time.Now()})

							} else {

								v.UUID = uuid
								v.Status = "selling"
								v.Rate = rate
								v.Quantity = quantity
								v.Fee = ofee
								v.Total = total

								sellerr := c.Update(bson.M{"_id": v.Id}, &v)

								if err != nil {

									e := session.DB("v2").C("ErrorLog").With(session)
									e.Insert(&db.ErrorLog{Description: "Place sell order", Error: sellerr.Error(), Time: time.Now()})

								}
							}*/
						}
					}
				}

			} /*else if time.Now().Sub(v.Buy.CompleteTime ).Hours() > 24 {
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
					e.Insert(&db.ErrorLog{Description:"24 hours Place sell Limit - API", Error:placeOErr.Error(), Time:time.Now()})

				}else {
					v.Sell.UUID = uuid
					v.Sell.Status = "selling"
					v.Status = "selling"
					v.Sell.Rate = rate
					v.Sell.Quantity = quantity
					v.Sell.Fee = ofee
					v.Sell.Total = total

					sellerr := c.Update(bson.M{"_id": v.Id}, &v)

					if err != nil {

						e := session.DB("v2").C("ErrorLog").With(session)
						e.Insert(&db.ErrorLog{Description: "Place sell order", Error: sellerr.Error(), Time: time.Now()})

					}
				}

			}*/

		}
		JobChannel <- t
	}
}