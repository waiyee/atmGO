package main

import (
	"fmt"
	"atm/bittrex"
	"gopkg.in/mgo.v2"
//	"gopkg.in/mgo.v2/bson"
	"log"
	"time"
//	"sync"
	"os"
	"os/signal"
	"syscall"
)

const (
	API_KEY    = "b16a576da6ce4c98a228a9c8fb358a9d"
	API_SECRET = "311cf541a233410c8f8d84d1a0e03d96"
)

// Receives the change in the number of goroutines
var goroutineDelta = make(chan int)
/*
func getMarkets()(markets []bittrex.Market){

	session, err := mgo.Dial("localhost:27017")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	c := session.DB("v2").C("Markets")

		err = c.Find(bson.M{ "$or": []bson.M{ bson.M{"marketname":"BTC-LTC"}, bson.M{"marketname": "BTC-GLD"} } }).All(&markets)
	if err != nil {
		log.Fatal(err)
	}


	return
} */

func periodicGetSummaries(t time.Time){

	b := bittrex.New(API_KEY, API_SECRET)

	marketSummaries, err := b.GetMarketSummaries()

	session, err := mgo.Dial("localhost:27017")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	c := session.DB("v2").C("MarketsSummaries")
	k := time.Now()
	fmt.Println("Got Market Summaries preparing insert ", k)
	for _,v:= range marketSummaries {
		v.DBTime = k
		err = c.Insert(&v)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// Conceptual code
func loopGetSummary() {
	fmt.Println("Inside Forver")
	/** Keep calling periodicGetSummaries every second in async mode **/
	for t := range time.NewTicker(time.Millisecond * 1000 ).C {
		go periodicGetSummaries(t)
	}
}




func main() {
	// Bittrex client
	bittrex := bittrex.New(API_KEY, API_SECRET)

	// Buffer for calling bittrex API
	balances, err := bittrex.GetTicker("BTC-LTC")
	fmt.Println(err, balances)

	/* Code for listen Ctrl + C to stop the bot*/
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		os.Exit(1)
	}()
	/* Code for listen Ctrl + C to stop the bot*/

	/* async call a job to get summaries */
	go loopGetSummary()

	fmt.Println("Just a test it is running without waiting")





	/* a code for END to wait running program */
	for range goroutineDelta {
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
