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
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"regexp"
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

func ServerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	sysTime := resTime{time.Now().String()}
	json.NewEncoder(w).Encode(sysTime)
}

func StatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Initialize a session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)
	if err != nil {
		log.Fatalf("Got error initializing AWS: %s", err)
	}

	// Create DynamoDB client
	svc := dynamodb.New(sess)

	// Describe the table
	input := &dynamodb.DescribeTableInput{
		TableName: aws.String("asigdel-topstocks"),
	}

	result, err := svc.DescribeTable(input)
	if err != nil {
		log.Fatalf("Got error describing table: %s", err)
	}

	// Create response struct to be turned into JSON
	var statusResponse TableStatus
	statusResponse.Table = "asigdel-topstocks"
	statusResponse.Count = result.Table.ItemCount

	// JSON Response
	json.NewEncoder(w).Encode(statusResponse)
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

	var allResponse []Item

	scanErr := svc.ScanPages(&dynamodb.ScanInput{
		TableName: aws.String("akc-citybikes"),
	}, func(page *dynamodb.ScanOutput, last bool) bool {
		recs := []Item{}

		err := dynamodbattribute.UnmarshalListOfMaps(page.Items, &recs)
		if err != nil {
			panic(fmt.Sprintf("failed to unmarshal Dynamodb Scan Items, %v", err))
		}

		allResponse = append(allResponse, recs...)

		return true
	})

	if scanErr != nil {
		panic(fmt.Sprintf("Got error scanning DB, %v", scanErr))
	}

	json.NewEncoder(w).Encode(allResponse)
}

func SearchHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	searchDate := mux.Vars(r)["date"]

	proper, err := regexp.MatchString("^\\d{4}\\-(0[1-9]|1[012])\\-(0[1-9]|[12][0-9]|3[01])$", searchDate)

	if err != nil {
		log.Fatal(err)
	}

	if proper {
		w.WriteHeader(http.StatusOK)

		sess, err := session.NewSession(&aws.Config{
			Region: aws.String("us-east-1")},
		)
		if err != nil {
			log.Fatalf("Got error initializing AWS: %s", err)
		}

		svc := dynamodb.New(sess)

		filt := expression.Contains(expression.Name("Time"), searchDate)

		expr, err := expression.NewBuilder().WithFilter(filt).Build()
		if err != nil {
			log.Fatalf("Got error building expression: %s", err)
		}

		params := &dynamodb.ScanInput{
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
			FilterExpression:          expr.Filter(),
			ProjectionExpression:      expr.Projection(),
			TableName:                 aws.String("akc-citybikes"),
		}

		out, err := svc.Scan(params)

		if err != nil {
			log.Fatalf("Query API call failed: %s", err)
		}
		searchResponse := []Item{}
		err = dynamodbattribute.UnmarshalListOfMaps(out.Items, &searchResponse)
		if err != nil {
			panic(fmt.Sprintf("Failed to unmarshal Record, %v", err))
		}

		json.NewEncoder(w).Encode(searchResponse)
	} else {
		w.WriteHeader(http.StatusBadRequest)
		badMessage := "Search should be formatted with search?date=yyyy-mm-dd"
		json.NewEncoder(w).Encode(badMessage)
	}
}

func NewLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

func loggingMiddleware(next http.Handler) http.Handler {
	//	os.Setenv("LOGGLY_TOKEN", "e4a25bf2-e2cc-4771-95c8-b9a68c55bc11")
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
	r.HandleFunc("/asigdel/server", ServerHandler).Methods("GET")
	r.HandleFunc("/asigdel/all", AllHandler).Methods("GET")
	r.HandleFunc("/asigdel/status", StatusHandler).Methods("GET")
	r.HandleFunc("/asigdel/search", SearchHandler).Queries("date", "{date:.*}")
	wrappedRouter := loggingMiddleware(r)
	http.ListenAndServe(":3000", wrappedRouter)
}
