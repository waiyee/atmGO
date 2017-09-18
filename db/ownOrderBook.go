package db

import "time"

type Orders struct{
	Status string
	MarketName string
	CreatedAt time.Time
	UpdatedAt time.Time
	Buy OrderBook
	Sell OrderBook
}

type OrderBook struct{
	UUID string
	Status string
	Quantity float64
	Rate float64
	Fee float64
	Total float64
	OrderTime time.Time
	CompleteTime time.Time

}
