package main

import (
	"time"
	"fmt"
	"sync"
	"atm/bittrex"

	"atm/db"
)

var minTotal = float64(60000)

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

				VOI := float64(0)
				VOI = bidVol - askVol
				OIR := float64(0)
				OIR = (bidVol - askVol) / (bidVol + askVol)
				Spread := orderBook.Sell[0].Rate - orderBook.Buy[0].Rate
				MPB := MMPB.Markets[markets[i]]

				final := (VOI / Spread) + (OIR / Spread ) + (MPB / Spread)

				session := mydb.Session.Clone()
				defer session.Close()
				c := session.DB("v2").C("OwnOrderBook").With(session)

				/** final may need to adjust to obtain a better result
					e.g. (final / 10000000) > 0.2
					need to test
				*/
				if final > 0{
					// place buy order at ask rate
					rate := orderBook.Sell[0].Rate
					// Attention for not enough balance?
					quantity := minTotal * (1-fee) / rate
					uuid := "xyz" // get from bittrex api
					buyorder := &db.Orders{
						Status : "buying",
						MarketName : markets[i],
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
						Buy: db.OrderBook{
							UUID: uuid,
							Status: "buying",
							Quantity: quantity,
							Rate: rate,
							OrderTime: time.Now(),
						},
					}
					err := c.Insert(&buyorder)
					if err != nil {
						fmt.Println("Place buy order - ", time.Now(), err)
					}
				}else if final < 0{
					// if stocks on hand
					// place sell order at bid rate
					//rate := orderBook.Buy[0].Rate
				}


				fmt.Printf("Market: %v , VOI: %f, OIR: %f, MPB: %f, Spread: %f, Final : %f \n", markets[i],VOI,OIR,MPB,Spread,final)




			}
			//defer wgm.Done()
			wg.Done()
		}(wg, i)

	}

	wg.Wait()


}



func refreshOrder(){
	for t:= range time.NewTicker(time.Millisecond * 500 ).C{
		// Prepare to update order status from bittrex
		bapi := bittrex.New(API_KEY, API_SECRET)
		orderHistory, err := bapi.GetOrderHistory("all")
		fmt.Println(err, orderHistory)

		JobChannel <- t
	}
}