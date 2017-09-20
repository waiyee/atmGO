package db

import "time"

type ErrorLog struct{
	Description string
	Error string
	Time time.Time

}