package main

import (
	"fmt"
	"atm/bittrex"
	"time"
	"os"
	"os/signal"
//	"atm/calculation"
	"syscall"
	"sync"
	"atm/db"

)




var mydb db.Mydbset
var BTCMarkets []string

// Receives the change in the number of goroutines
var JobChannel = make(chan time.Time)

type thisSecondMarket struct {
	Markets map[string]bittrex.MarketSummary
	Lock sync.Mutex
}

type lastSecondMarket struct {
	Markets map[string]bittrex.MarketSummary
	Lock sync.Mutex
}

type MarketMPB struct {
	Markets map[string]float64
	Lock sync.Mutex
}

var thisSM thisSecondMarket
var lastSM lastSecondMarket
var MMPB MarketMPB


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
				//c := session.DB("v2").C("OwnOrderBook").With(session)

				if final > 0{
					// place buy order at ask rate
				}else{
					// place buy order at sell rate
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
	for t:= range time.NewTicker(time.Second * 20).C{
		// Prepare to update order status from bittrex
		JobChannel <- t
	}
}


func loopGetOrderBook()  {
/*	var wgo sync.WaitGroup
	wgo.Add(1)*/
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
		//defer wgo.Done()
	}
	//wgo.Wait()
}

/**
 * Used to refresh the available markets in bittrex, once per program should good enough
 */
func refreshMarkets(){
	bAPI := bittrex.New(API_KEY, API_SECRET)

	markets, err := bAPI.GetMarkets()
	if err != nil {
		fmt.Println("refreshMarkets - " , time.Now(), err)
	}

	i := 0
	for _,v := range markets {
		if v.BaseCurrency == "BTC"{
			BTCMarkets = append(BTCMarkets, v.MarketName)
			i++
		}
	}
}

func main() {

	mydb = db.NewDbSession("mongodb://localhost:27017/?authSource=v2", "v2")

	thisSM.Markets = make(map[string]bittrex.MarketSummary)
	lastSM.Markets = make(map[string]bittrex.MarketSummary)
	MMPB.Markets = make(map[string]float64)
	// Bittrex client
	//bAPI := bittrex.New(API_KEY, API_SECRET)

	// Buffer for calling bittrex API
	/*balances, err := bAPI.GetTicker("BTC-LTC")
	fmt.Println(time.Now(),err, balances)*/

	refreshMarkets()


	/* Code for listen Ctrl + C to stop the bot*/
	cc := make(chan os.Signal, 1)
	signal.Notify(cc, os.Interrupt, syscall.SIGTERM)
	go func() {
		for range cc {
			mydb.Session.Close()
			close(cc)
			close(JobChannel)
			os.Exit(1)
		}

	}()
	/* Code for listen Ctrl + C to stop the bot*/

	/* async call a job to get summaries */
	go loopGetSummary()

	go loopGetOrderBook()

	go refreshOrder()


	for j:= range JobChannel{
		fmt.Println("Worked ", j )

	}

	/* a code for END to wait running program */
	/*for {
	}
	/* a code for END to wait running program */

	// Get Candle ( OHLCV )

	/*
	markets, err := bittrex.GetHisCandles("BTC-LTC", "hour")
		fmt.Println(markets, err)
	markets, err := bittrex.GetMarkets()
	*/

	// Get markets
/*	fmt.Println(time.Now())
	markets := getMarkets()
	var wg sync.WaitGroup
	wg.Add(len(markets))

		for i := 0; i < len(markets); i++ {
			go func(i int) {
				defer wg.Done()
				fmt.Println(i, markets[i].MarketName, time.Now(), " START")
				//time.Sleep(10000 * time.Millisecond)
				ticker, err := bittrex.GetMarketSummary(markets[i].MarketName)
				fmt.Println(i, markets[i].MarketName, time.Now(), ticker, err, " END")

			}(i)

		}

	wg.Wait()*/


	//	go forever()

/*
	numGoroutines := 0

	for diff := range goroutineDelta {
		numGoroutines += diff
		if numGoroutines == 0 { fmt.Println("test")}
	}


*/


/*
	fmt.Println(time.Now())
	balances, err := bittrex.GetTicker("BTC-LTC")
	fmt.Println(err, balances)
	fmt.Println(time.Now())
	//markets := getMarkets()
	fmt.Println( "BTC-LTC", time.Now(), " START")
	ticker, err := bittrex.GetMarketSummary("BTC-LTC")
	fmt.Println("BTC-LTC", time.Now(), ticker, err, " END")
	fmt.Println("get market summaries", time.Now(), ticker, err, " START")

	marketSummaries, err := bittrex.GetMarketSummaries()
	//fmt.Println(err, marketSummaries)
	fmt.Println("get market summaries", time.Now(), marketSummaries, err, " END")

	/*for i := 0; i < 20 ; i++ {
		fmt.Println(i, "BTC-LTC", time.Now(), " START")
		ticker, err := bittrex.GetMarketSummary("BTC-LTC")
		fmt.Println(i,"BTC-LTC", time.Now(), ticker, err, " END")
		time.Sleep(time.Second)
	}
*/

	// Get Ticker (BTC-VTC)
	/*
		ticker, err := bittrex.GetTicker("BTC-DRK")
		fmt.Println(err, ticker)
	*/

	// Get Distribution (JBS)
	/*
		distribution, err := bittrex.GetDistribution("JBS")
		for _, balance := range distribution.Distribution {
			fmt.Println(balance.BalanceD)
		}
	*/

	// Get market summaries
	/*
		marketSummaries, err := bittrex.GetMarketSummaries()
		fmt.Println(err, marketSummaries)
	*/

	// Get market summary
	/*
		marketSummary, err := bittrex.GetMarketSummary("BTC-ETH")
		fmt.Println(err, marketSummary)
	*/

	// Get orders book
	/*
		orderBook, err := bittrex.GetOrderBook("BTC-DRK", "both", 100)
		fmt.Println(err, orderBook)
	*/

	// Get order book buy or sell side only
	/*
		orderb, err := bittrex.GetOrderBookBuySell("BTC-JBS", "buy", 100)
		fmt.Println(err, orderb)
	*/

	// Market history
	/*
		marketHistory, err := bittrex.GetMarketHistory("BTC-DRK", 100)
		for _, trade := range marketHistory {
			fmt.Println(err, trade.Timestamp.String(), trade.Quantity, trade.Price)
		}
	*/

	// Market

	// BuyLimit
	/*
		uuid, err := bittrex.BuyLimit("BTC-DOGE", 1000, 0.00000102)
		fmt.Println(err, uuid)
	*/

	// BuyMarket
	/*
		uuid, err := bittrex.BuyLimit("BTC-DOGE", 1000)
		fmt.Println(err, uuid)
	*/

	// Sell limit
	/*
		uuid, err := bittrex.SellLimit("BTC-DOGE", 1000, 0.00000115)
		fmt.Println(err, uuid)
	*/

	// Cancel Order
	/*
		err := bittrex.CancelOrder("e3b4b704-2aca-4b8c-8272-50fada7de474")
		fmt.Println(err)
	*/

	// Get open orders
	/*
		orders, err := bittrex.GetOpenOrders("BTC-DOGE")
		fmt.Println(err, orders)
	*/

	// Account
	// Get balances
	/*
		balances, err := bittrex.GetBalances()
		fmt.Println(err, balances)
	*/

	// Get balance
	/*
		balance, err := bittrex.GetBalance("DOGE")
		fmt.Println(err, balance)
	*/

	// Get address
	/*
		address, err := bittrex.GetDepositAddress("QBC")
		fmt.Println(err, address)
	*/

	// WithDraw
	/*
		whitdrawUuid, err := bittrex.Withdraw("QYQeWgSnxwtTuW744z7Bs1xsgszWaFueQc", "QBC", 1.1)
		fmt.Println(err, whitdrawUuid)
	*/

	// Get order history
	/*
		orderHistory, err := bittrex.GetOrderHistory("BTC-DOGE", 10)
		fmt.Println(err, orderHistory)
	*/

	// Get getwithdrawal history
	/*
		withdrawalHistory, err := bittrex.GetWithdrawalHistory("all", 0)
		fmt.Println(err, withdrawalHistory)
	*/

	// Get deposit history
	/*
		deposits, err := bittrex.GetDepositHistory("all", 0)
		fmt.Println(err, deposits)
	*/

}
