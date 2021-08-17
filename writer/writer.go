package main

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/edgelesssys/ego/marble"
	"github.com/go-sql-driver/mysql"
)

// Entry holds the information we gather from the API and add to the database
type Entry struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

func main() {
	// Setup required MySQL TLS connection with MarbleRun as trust anchor
	db, err := setupDatabaseConnection()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Disable HTTPS verification for external API
	// This is not good for production use, however EGo does not provide "trusted" certificates so far for external connections
	// Therefore, we disable this just for HTTP
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// Every 10 seconds, get some random user data from random-data-api.com and add it to the database
	for range time.Tick(time.Second * 10) {
		log.Println("Collecting data from Random Data API...")
		resp, err := http.Get("https://random-data-api.com/api/users/random_user")
		if err != nil {
			log.Println("ERROR: ", err)
			continue
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println("ERROR: ", err)
			continue
		}
		resp.Body.Close()
		var data Entry
		if err := json.Unmarshal(body, &data); err != nil {
			log.Println("ERROR: ", err)
			continue
		}

		log.Println("Inserting into database:", data)
		res, err := db.Query("INSERT INTO data VALUES (NULL, ?, ?, ?)", data.FirstName, data.LastName, data.Email)
		if err != nil {
			log.Println("ERROR: ", err)
			continue
		}
		res.Close()
	}
}

// setupDatabaseConnection setups the connection to EdgelessDB, based on MarbleRun's root CA certificate
func setupDatabaseConnection() (*sql.DB, error) {
	log.Println("Attempting to connect to database...")
	rootCertPool := x509.NewCertPool()
	pem := []byte(os.Getenv(marble.MarbleEnvironmentRootCA))
	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		return nil, errors.New("could not add root cert from PEM")
	}
	// Get certificate + private key for writer instance
	cert, err := tls.X509KeyPair([]byte(os.Getenv("CERT")), []byte(os.Getenv("KEY")))
	if err != nil {
		return nil, err
	}

	mysql.RegisterTLSConfig("edgelessdb", &tls.Config{
		RootCAs:      rootCertPool,
		Certificates: []tls.Certificate{cert},
	})

	dbHost := os.Getenv("EDG_DB_HOST")
	if dbHost == "" {
		dbHost = "localhost:3306"
	}
	dsn := "writer@tcp(" + dbHost + ")/users?tls=edgelessdb"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	} else if err := db.Ping(); err != nil {
		return nil, err
	}
	log.Println("Connected to database!")
	return db, nil
}
