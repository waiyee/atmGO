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



type topOrderMarkets struct {
	MarketName      string    `json:"marketname" bson:"marketname"`
	Vol      float64          `json:"vol" bson:"vol"`
}

var mydb db.Mydbset


var JobChannel = make(chan time.Time)
//var a int64
// Receives the change in the number of goroutines
//var goroutineDelta = make(chan int)

func getMarkets()(markets []topOrderMarkets){

	markets = []topOrderMarkets{topOrderMarkets{"BTC-ETH", 0},
		topOrderMarkets{"BTC-NEO", 0},
		topOrderMarkets{"BTC-OMG", 0},
		topOrderMarkets{"BTC-ETC", 0},
		topOrderMarkets{"BTC-TRIG", 0},
		topOrderMarkets{"BTC-QTUM", 0},
		topOrderMarkets{"BTC-OK", 0},
		topOrderMarkets{"BTC-LTC", 0},
		topOrderMarkets{"BTC-ZEN", 0},
		}

/*	session, err := mgo.Dial("localhost:27017")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)
*/
/*	session := mydb.Session.Clone()
	defer session.Close()
	c := session.DB("v2").C("MarketsLatest").With(session)
	query := []bson.M{
		{
			"$match": bson.M{
				"marketname" :  bson.RegEx{`^BTC-`, ""},
			},
		},
		{
			"$group" : bson.M{
				"_id" : "$marketname",
				"maxTime" : bson.M{"$max" : "$dbtime"},
				"doc" : bson.M{"$last" : "$$ROOT"},
			},
		},
		{
			"$sort" : bson.M{
				"doc.basevolume" : -1,
			},
		},
		{
			"$project" : bson.M{
				"_id" : 0,
				"marketname" : "$doc.marketname",
				"vol" : "$doc.basevolume",
			},
		},

		{
			"$limit" : 8,
		},
	}

	pipe := c.Pipe(query)
	err :=pipe.All(&markets)
	if err != nil {
		fmt.Println("getMarkets", time.Now(), err)
	}

	c.DropCollection()*/
	return
}
/*
func periodicGetSummaries(t time.Time){

	b := bittrex.New(API_KEY, API_SECRET)

	marketSummaries, err := b.GetMarketSummaries()

	/*session, err := mgo.Dial("localhost:27017")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)*/
	/*session := mydb.Session.Clone()
	defer session.Close()
	c := session.DB("v2").C("MarketsSummaries").With(session)
	d := session.DB("v2").C("MarketsLatest").With(session)
	k := time.Now()
	//fmt.Println("Got Market Summaries preparing insert : ", k)
	for _,v:= range marketSummaries {
		v.DBTime = k
		//err = c.Insert(&v)
		//err2 := d.Insert(&v)
		if err != nil {
			fmt.Println("periodicGetSummaries", time.Now(), err)
		}
		/*if err2!= nil {
			fmt.Println("periodicGetSummaries2", time.Now(), err2)
		}
	}
}
*/
var thisSecondMarket map[string]bittrex.MarketSummary
var lastSecondMarket map[string]bittrex.MarketSummary

func CompareMarkets(){
	for _,v := range thisSecondMarket{
		if v.BaseVolume != lastSecondMarket[v.MarketName].BaseVolume || v.Volume != lastSecondMarket[v.MarketName].Volume {
			fmt.Println("call order book")
		}else {
			fmt.Println("don't call order book")
		}

	}
	lastSecondMarket = thisSecondMarket
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

			for _,v:= range markets {
				thisSecondMarket[v.MarketName] = v
			}


			if len(lastSecondMarket) == 0 {
				fmt.Println(1)
				lastSecondMarket = thisSecondMarket
			}else{
				fmt.Println(2)
				CompareMarkets()
			}

		}()

		JobChannel <- t
	}
}


func periodicGetOrderBook(t time.Time)  {
	wg := &sync.WaitGroup{}
	markets := getMarkets()


	obRate := 0.125
/*	var ak string
	var as string
	if a%2  == 1 {
		ak = API_KEY
		as = API_SECRET
	} else{
		ak = API_KEY2
		as = API_SECRET2
	}
	a++*/
	bittrex := bittrex.New(API_KEY, API_SECRET)

	for i := 0; i < len(markets); i++ {
		wg.Add(1)
		go func(wg *sync.WaitGroup, i int) {

			orderBook, err := bittrex.GetOrderBook(markets[i].MarketName, "both")
			if err != nil {
				fmt.Println("periodicGetOrderBook1 - ", time.Now(), err)
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
			//fmt.Println("Order - " ,time.Now(), markets[i].MarketName, midPrice, quBidRate, quAskRate, bidVol, askVol)
				// 	session, err := mgo.Dial("localhost:27017")
			// if err != nil {
			//	panic(err)
			//}
			//defer session.Close()
			// Optional. Switch the session to a monotonic behavior.
			//session.SetMode(mgo.Monotonic, true)
				/*session := mydb.Session.Clone()
				defer session.Close()
				c := session.DB("v2").C("OrderBookStream").With(session)


				err = c.Insert(&OrderBookStream{time.Now(), markets[i].MarketName, midPrice, quBidRate, quAskRate, bidVol, askVol})

				if err != nil {
					fmt.Println("periodicGetOrderBook - " , time.Now(), err)
				}
*/
				//fmt.Println(i, " - ", time.Now(), markets[i].MarketName, " - Order Book Stream Finish Calculation")

			}
			//defer wgm.Done()
			wg.Done()
		}(wg, i)

	}

	wg.Wait()


}


func loopGetOrderBook()  {
/*	var wgo sync.WaitGroup
	wgo.Add(1)*/
	for t := range time.NewTicker(time.Second ).C {

		go periodicGetOrderBook(t)
		JobChannel <- t
		//defer wgo.Done()
	}
	//wgo.Wait()
}



func main() {
	mydb = db.NewDbSession("mongodb://localhost:27017/?authSource=v2", "v2")

	thisSecondMarket = make(map[string]bittrex.MarketSummary)
	lastSecondMarket = make(map[string]bittrex.MarketSummary)
	// Bittrex client
	bittrex := bittrex.New(API_KEY, API_SECRET)

	// Buffer for calling bittrex API
	balances, err := bittrex.GetTicker("BTC-LTC")
	fmt.Println(time.Now(),err, balances)

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
