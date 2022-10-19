package main

import (
	"github.com/JamesPEarly/loggly"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
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

func goDotEnvVariable(key string) string {
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	return os.Getenv(key)
}

func main() {
	os.Setenv("LOGGLY_TOKEN", goDotEnvVariable("LOGGLY_TOKEN"))
	os.Setenv("AWS_ACCESS_KEY_ID", goDotEnvVariable("AWS_ACCESS_KEY_ID"))
	os.Setenv("AWS_SECRET_ACCESS_KEY", goDotEnvVariable("AWS_SECRET_ACCESS_KEY"))
	r := mux.NewRouter()
	wrappedRouter := loggingMiddleware(r)
	http.ListenAndServe(":3000", wrappedRouter)
}
