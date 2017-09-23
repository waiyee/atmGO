package tradeHelper

import (
	"atm/db"
	"time"
	"gopkg.in/mgo.v2/bson"
	"atm/bittrex"
	"strings"
)

var fee float64 = 0.0025

func BuyHelper(rate float64, quantity float64, market string, btcbalance float64, final float64, bapi bittrex.Bittrex, mydb db.Mydbset, remark string){
	session := mydb.Session.Clone()
	defer session.Close()
	//uuid := "xyz" // get from bittrex api
	uuid,placeOErr := bapi.BuyLimit(market,quantity, rate)
	if placeOErr!= nil{
		e := session.DB("v2").C("ErrorLog").With(session)
		e.Insert(&db.ErrorLog{Description:"Place Buy Limit - API", Error:placeOErr.Error(), Time:time.Now()})

	} else {
		ofee := rate * quantity * fee
		total := ( rate * quantity ) + ofee
		wallet := btcbalance - total
		buyorder := &db.Orders{
			Status:      "buying",
			MarketName:  market,
			CurrentRate: rate,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			UUID:      uuid,
			Quantity:  quantity,
			Rate:      rate,
			Fee:       ofee,
			Total:     total,
			OrderTime: time.Now(),
			Final:     final,
			Remark:		remark,
		}
		c := session.DB("v2").C("OwnOrderBook2").With(session)
		d := session.DB("v2").C("WalletBalance").With(session)
		err := c.Insert(&buyorder)
		err2 := d.Update(bson.M{"currency": "BTC"}, bson.M{"$set": bson.M{"available": wallet}})
		err3 := d.Update(bson.M{"currency": strings.Split(market, "-")[1]}, bson.M{"$set": bson.M{"available": 0}})
		if err != nil {
			e := session.DB("v2").C("ErrorLog").With(session)
			e.Insert(&db.ErrorLog{Description: "Place buy order", Error: err.Error(), Time: time.Now()})

		}
		if err2 != nil {
			e := session.DB("v2").C("ErrorLog").With(session)
			e.Insert(&db.ErrorLog{Description: "Update Wallet BTC Balance @ buy order", Error: err2.Error(), Time: time.Now()})
		}
		if err3 != nil && err3.Error() != "not found"{
			e := session.DB("v2").C("ErrorLog").With(session)
			e.Insert(&db.ErrorLog{Description: "Update Wallet Market Balance @ buy order", Error: err3.Error(), Time: time.Now()})
		}/*else {

			err4 := d.Insert(&bittrex.Balance{Currency:strings.Split(market, "-'")[1], Available : quantity})
			if err4 != nil{
				e := session.DB("v2").C("ErrorLog").With(session)
				e.Insert(&db.ErrorLog{Description: "Insert Wallet Market Balance @ buy order", Error: err4.Error(), Time: time.Now()})
			}
		}*/
	}
}

func SellHelper (rate float64, quantity float64, market string, btcbalance float64, final float64, bapi bittrex.Bittrex, mydb db.Mydbset, remark string){
	session := mydb.Session.Clone()
	defer session.Close()
	//uuid := "abc"
	uuid,placeOErr := bapi.SellLimit(market,quantity, rate)
	if placeOErr!= nil{
		e := session.DB("v2").C("ErrorLog").With(session)
		e.Insert(&db.ErrorLog{Description:"Place Buy Limit - API", Error:placeOErr.Error(), Time:time.Now()})
	}else {
		ofee := (rate * quantity) * fee
		total := (rate * quantity) - ofee
		wallet := btcbalance + total
		sellorder := &db.Orders{
			Status:      "selling",
			MarketName:  market,
			CurrentRate: rate,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			UUID:      uuid,
			Quantity:  quantity,
			Rate:      rate,
			Fee:       ofee,
			Total:     total,
			OrderTime: time.Now(),
			Final:     final,
			Remark:		remark,
		}

		c := session.DB("v2").C("OwnOrderBook2").With(session)
		d := session.DB("v2").C("WalletBalance").With(session)
		err := c.Insert(&sellorder)
		err2 := d.Update(bson.M{"currency": "BTC"}, bson.M{"$set": bson.M{"available": wallet}})
		err3 := d.Update(bson.M{"currency": strings.Split(market, "-")[1]}, bson.M{"$set": bson.M{"available": 0}})
		if err != nil {

			e := session.DB("v2").C("ErrorLog").With(session)
			e.Insert(&db.ErrorLog{Description: "Place sell order", Error: err.Error(), Time: time.Now()})

		}
		if err2 != nil {
			e := session.DB("v2").C("ErrorLog").With(session)
			e.Insert(&db.ErrorLog{Description: "Update Wallet BTC Balance @ sell order", Error: err2.Error(), Time: time.Now()})
		}
		if err3 != nil {
			e := session.DB("v2").C("ErrorLog").With(session)
			e.Insert(&db.ErrorLog{Description: "Update Wallet Market Balance @ sell order", Error: err3.Error(), Time: time.Now()})
		}
	}
}