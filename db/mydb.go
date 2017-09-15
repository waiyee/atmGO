package db

import (

	"os"
	"gopkg.in/mgo.v2"

	"time"
)
type Mydbset struct{
	Session *mgo.Session
	Database *mgo.Database
}

/*
type DbLink struct {
	DbName string
}
*/

func NewDbSession(url string, db string)(r Mydbset) {
	var e error

	var s  *mgo.Session
	var d *mgo.Database

	s, e = mgo.Dial(url)
	d = s.DB(db)

	s.SetMode(mgo.Monotonic, true)
	s.SetSocketTimeout(5 * time.Minute)

	r = Mydbset{s,d}

	if e != nil {
		panic(e)
		os.Exit(-1)
	}
	return
}
