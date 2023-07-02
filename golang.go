package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type Spot struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Location    string  `json:"location"`
	Website     string  `json:"website"`
	Description string  `json:"description"`
	Rating      float64 `json:"rating"`
	Distance    float64
}

type Spots []Spot

func spotsInAreaHandler(w http.ResponseWriter, r *http.Request) {
	var latitude, longitude, radius float64
	var isCircle bool

	latStr := r.URL.Query().Get("latitude")
	longStr := r.URL.Query().Get("longitude")
	radiusStr := r.URL.Query().Get("radius")
	isCircleStr := r.URL.Query().Get("isCircle")

	// Parse request parameters
	latitude, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		log.Fatal(err)
	}
	longitude, err = strconv.ParseFloat(longStr, 64)
	if err != nil {
		log.Fatal(err)
	}
	radius, err = strconv.ParseFloat(radiusStr, 64)
	if err != nil {
		log.Fatal(err)
	}

	// Determine if the shape is a circle or square
	isCircleStr = strings.ToLower(isCircleStr)
	if isCircleStr != "circle" && isCircleStr != "square" {
		log.Fatal("Invalid value for isCircle parameter")
	}
	isCircle = (isCircleStr == "circle")

	// Load environment variables from .env file
	err = godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Database configuration
	dbHost := os.Getenv("DB_HOST")
	dbPort, err := strconv.Atoi(os.Getenv("DB_PORT"))
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")

	dbInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", dbHost, dbPort, dbUser, dbPassword, "Spots")
	db, err := sql.Open("postgres", dbInfo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var query string
	if isCircle {
		// Query for spots within a circle
		query = fmt.Sprintf("SELECT * FROM (SELECT id, name, COALESCE(website, ''), coordinates, COALESCE(description, ''), rating, ST_Distance(coordinates::geography, 'SRID=4326;POINT(%f %f)'::geography) AS distance FROM public.\"MY_TABLE\" WHERE ST_DWithin(coordinates, 'SRID=4326;POINT(%f %f)'::geography, %f)) AS subquery;", longitude, latitude, longitude, latitude, radius)
	} else {
		// Calculate the coordinates of the square and query for spots within it
		bottomLeftLongitude, bottomLeftLatitude, topRightLongitude, topRightLatitude := calculateSquareCoordinates(longitude, latitude, radius)
		query = fmt.Sprintf("SELECT id, name, COALESCE(website, ''), coordinates, COALESCE(description, ''), rating, ST_Distance(coordinates, 'SRID=4326;POINT(%f %f)'::geography) AS distance FROM public.\"MY_TABLE\" WHERE ST_DWithin(coordinates, ST_MakeEnvelope(%f, %f, %f, %f, 4326), 0);", longitude, latitude, bottomLeftLongitude, bottomLeftLatitude, topRightLongitude, topRightLatitude)
	}

	queryResult, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer queryResult.Close()

	var spotList []Spot
	for queryResult.Next() {
		var row Spot
		err := queryResult.Scan(&row.ID, &row.Name, &row.Location, &row.Website, &row.Description, &row.Rating, &row.Distance)
		if err != nil {
			log.Fatal(err)
		}
		spotList = append(spotList, row)
	}

	// Sort the spots based on distance and rating
	sort.Slice(spotList, func(i, j int) bool {
		if math.Abs(spotList[i].Distance-spotList[j].Distance) < 50 {
			return spotList[i].Rating > spotList[j].Rating
		} else {
			return spotList[i].Distance < spotList[j].Distance
		}
	})

	// Print spot information
	for _, spot := range spotList {
		fmt.Printf("Spot: Name=%s, Distance=%.2f, Rating=%.2f\n", spot.Name, spot.Distance, spot.Rating)
	}

	// Encode and send the spots as JSON response
	json.NewEncoder(w).Encode(Spots(spotList))
}

func main() {
	// Create a new router
	router := mux.NewRouter()

	// Define the endpoint route and handler
	router.HandleFunc("/spots-in-area", spotsInAreaHandler).Methods("GET")

	// Start the HTTP server
	log.Println("Server listening on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func calculateSquareCoordinates(centerLongitude, centerLatitude, meters float64) (bottomLeftLong, bottomLeftLat, topRightLong, topRightLat float64) {
	degreesPerMeter := 1 / 111319.9
	degrees := meters * degreesPerMeter

	// Calculate bottom left and top right coordinates of the square
	bottomLeftLongitude := centerLongitude - degrees
	bottomLeftLatitude := centerLatitude - degrees
	topRightLongitude := centerLongitude + degrees
	topRightLatitude := centerLatitude + degrees

	return bottomLeftLongitude, bottomLeftLatitude, topRightLongitude, topRightLatitude
}
