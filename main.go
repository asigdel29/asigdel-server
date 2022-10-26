package main

import (
	"encoding/json"
	"fmt"
	"github.com/JamesPEarly/loggly"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

type TableStatus struct {
	Table string `json:"table"`
	Count *int64 `json:"recordCount"`
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

	var status TableStatus
	status.Table = "asigdel-topstocks"
	status.Count = result.Table.ItemCount

	json.NewEncoder(w).Encode(status)
}

func AllHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)
	if err != nil {
		log.Fatalf("Got error initializing AWS: %s", err)
	}

	svc := dynamodb.New(sess)

	var all []Summary
	scanErr := svc.ScanPages(&dynamodb.ScanInput{
		TableName: aws.String("asigdel-topstocks"),
	}, func(page *dynamodb.ScanOutput, last bool) bool {
		recs := []Summary{}

		err := dynamodbattribute.UnmarshalListOfMaps(page.Items, &recs)
		if err != nil {
			panic(fmt.Sprintf("failed to unmarshal Dynamodb Items, %v", err))
		}
		all = append(all, recs...)
		return true
	})

	if scanErr != nil {
		panic(fmt.Sprintf("Got error scanning DB, %v", scanErr))
	}
	json.NewEncoder(w).Encode(all)
}

func SearchHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	search := mux.Vars(r)["date"]
	check, err := regexp.MatchString("^\\d{4}\\-(0?[1-9]|1[012])\\-(0?[1-9]|[12][0-9]|3[01])$", search)

	if err != nil {
		log.Fatal(err)
	}

	if check {
		w.WriteHeader(http.StatusOK)
		sess, err := session.NewSession(&aws.Config{
			Region: aws.String("us-east-1")},
		)

		svc := dynamodb.New(sess)
		filt := expression.Contains(expression.Name("Time"), search)
		expr, err := expression.NewBuilder().WithFilter(filt).Build()
		if err != nil {
			log.Fatalf("Got error building expression: %s", err)
		}

		params := &dynamodb.ScanInput{
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			FilterExpression:          expr.Filter(),
			ProjectionExpression:      expr.Projection(),
			TableName:                 aws.String("asigdel-topstocks"),
		}

		out, err := svc.Scan(params)

		if err != nil {
			log.Fatalf("Query API call failed: %s", err)
		}
		search := []Summary{}
		err = dynamodbattribute.UnmarshalListOfMaps(out.Items, &search)
		if err != nil {
			panic(fmt.Sprintf("Failed to unmarshal Record, %v", err))
		}
		json.NewEncoder(w).Encode(search)
	} else {
		w.WriteHeader(http.StatusBadRequest)
		badMessage := "Search not formatted correctly,YYYY-MM-DD"
		json.NewEncoder(w).Encode(badMessage)
	}
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
	r.HandleFunc("/asigdel/all", AllHandler).Methods("GET")
	r.HandleFunc("/asigdel/search", SearchHandler).Queries("date", "{date:.*}")
	wrappedRouter := loggingMiddleware(r)
	http.ListenAndServe(":8080", wrappedRouter)
}
