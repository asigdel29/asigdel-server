package main

import (
	"encoding/json"
	"github.com/JamesPEarly/loggly"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
	"time"
)

type TableStatus struct {
	Table string `json:"table"`
	Count *int64 `json:"recordCount"`
}

type resTime struct {
	SystemTime string
}

type Item struct {
	Summary     Summary
	StockSymbol string
	Time        string
}

type Response struct {
	Summary Summary `json:"Summary"`
}

type Summary struct {
	Name                 string    `json:"Name"`
	StockSymbol          string    `json:"StockSymbol"`
	Price                float64   `json:"Price"`
	DollarChange         float64   `json:"DollarChange"`
	PercentChange        float64   `json:"PercentChange"`
	PreviousClose        float64   `json:"PreviousClose"`
	Open                 float64   `json:"Open"`
	BidPrice             float64   `json:"BidPrice"`
	BidQuantity          int       `json:"BidQuantity"`
	AskPrice             float64   `json:"AskPrice"`
	AskQuantity          int       `json:"AskQuantity"`
	DayRangeLow          float64   `json:"DayRangeLow"`
	DayRangeHigh         float64   `json:"DayRangeHigh"`
	YearRangeLow         float64   `json:"YearRangeLow"`
	YearRangeHigh        float64   `json:"YearRangeHigh"`
	Volume               int       `json:"Volume"`
	AverageVolume        int       `json:"AverageVolume"`
	MarketCap            float64   `json:"MarketCap"`
	Beta                 float64   `json:"Beta"`
	PriceEarningsRatio   float64   `json:"PriceEarningsRatio"`
	EarningsPerShare     float64   `json:"EarningsPerShare"`
	EarningsDate         string    `json:"EarningsDate"`
	ForwardDividend      float64   `json:"ForwardDividend"`
	ForwardDividendYield float64   `json:"ForwardDividendYield"`
	ExDividendDate       string    `json:"ExDividendDate"`
	YearTargetEstimate   float64   `json:"YearTargetEstimate"`
	QueriedSymbol        string    `json:"QueriedSymbol"`
	DataCollectedOn      time.Time `json:"DataCollectedOn"`
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func StatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)
	if err != nil {
		log.Fatalf("Got error initializing AWS: %s", err)
	}

	svc := dynamodb.New(sess)

	input := &dynamodb.DescribeTableInput{
		TableName: aws.String("asigdel-topstocks"),
	}

	result, err := svc.DescribeTable(input)
	if err != nil {
		log.Fatalf("Got error describing table: %s", err)
	}

	var statusResponse TableStatus
	statusResponse.Table = "asigdel-topstocks"
	statusResponse.Count = result.Table.ItemCount

	json.NewEncoder(w).Encode(statusResponse)
}

func NewLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lrw := NewLoggingResponseWriter(w)
		next.ServeHTTP(lrw, r)

		var tag string = "server"
		client := loggly.New(tag)
		client.EchoSend("info", "Method type: "+r.Method+" | Source IP address: "+r.RemoteAddr+" | Request Path: "+r.RequestURI+" | Status Code: "+strconv.Itoa(lrw.statusCode))
	})
}

func main() {

	r := mux.NewRouter()
	r.HandleFunc("/asigdel/status", StatusHandler).Methods("GET")
	wrappedRouter := loggingMiddleware(r)
	http.ListenAndServe(":3000", wrappedRouter)
}
