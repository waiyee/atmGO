package db

import (
	"time"
	"gopkg.in/mgo.v2/bson"
)

type Orders struct{
	Id bson.ObjectId  `json:"id"        bson:"_id,omitempty"`
	Status string
	MarketName string
	CurrentRate float64
	CreatedAt time.Time
	UpdatedAt time.Time
	UUID string
	Quantity float64
	Rate float64
	Fee float64
	Total float64
	OrderTime time.Time
	CompleteTime time.Time
	Final float64
	Remark string
}


