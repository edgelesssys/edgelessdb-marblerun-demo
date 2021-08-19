package main

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/edgelesssys/ego/marble"
	"github.com/go-sql-driver/mysql"
)

const htmlPage = `
<!DOCTYPE html>
<html>

<head>
    <style>
        table,
        th,
        td {
            border: 1px solid black;
            border-collapse: collapse;
        }

        th,
        td {
            padding: 5px;
        }

        td {
            text-align: center;
        }
    </style>
    <title>User Database</title>
</head>

<body>
    <table>
        <tr>
            <th>ID</th>
            <th>First Name</th>
            <th>Last Name</th>
            <th>E-Mail Address</th>
        </tr>
        {{range .}}
        <tr>
            <td>{{.ID}}</td>
            <td>{{.FirstName}}</td>
            <td>{{.LastName}}</td>
            <td>{{.Email}}</td>
        </tr>
        {{end}}
    </table>
</body>

</html>
`

// Entry holds the information the database returns. Similar to the one in writer go, just with the additional ID for each entry in the database.
type Entry struct {
	ID        int
	FirstName string
	LastName  string
	Email     string
}

func main() {
	// Setup SQL TLS connection
	log.Println("Setting up TLS connection parameters...")
	if err := setupDatabaseTLS(); err != nil {
		panic(err)
	}

	// Initialize HTTP server
	http.HandleFunc("/", handler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	cert, err := tls.X509KeyPair([]byte(os.Getenv("CERT")), []byte(os.Getenv("KEY")))
	if err != nil {
		log.Fatal(err)
	}
	server := http.Server{
		Addr: ":" + port,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
	}

	log.Printf("Listening on port %s", port)
	if err := server.ListenAndServeTLS("", ""); err != nil {
		log.Fatal(err)
	}
}

// handler is called whenever an user makes a connection to the HTTP server
// Here, we connect to the database and select everything from
func handler(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling connection...")

	// Connect to EdgelessDB
	log.Println("Connecting to the database...")
	dbHost := os.Getenv("EDG_DB_HOST")
	if dbHost == "" {
		dbHost = "localhost:3306"
	}
	dsn := "reader@tcp(" + dbHost + ")/users?tls=edgelessdb"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Println("ERROR: ", err)
		return
	} else if err := db.Ping(); err != nil {
		log.Println("ERROR: ", err)
		return
	}
	defer db.Close()

	// Retrieve all data from users.data
	log.Println("Connected, retrieving data...")
	res, err := db.Query("SELECT * FROM data")
	if err != nil {
		log.Println("ERROR: ", err)
		return
	}
	defer res.Close()
	var Results []Entry
	for res.Next() {
		var result Entry
		if err := res.Scan(&result.ID, &result.FirstName, &result.LastName, &result.Email); err != nil {
			log.Println("ERROR: ", err)
			return
		}
		Results = append(Results, result)
	}
	log.Println(Results)

	// Insert into HTML template
	t := template.New("index.html")
	t, err = t.Parse(htmlPage)
	if err != nil {
		log.Println("ERROR: ", err)
		return
	}

	// Display the results to the user
	err = t.Execute(w, Results)
	if err != nil {
		log.Println("ERROR: ", err)
		return
	}
}

// setupDatabaseTLS sets up the TLS configuration used when establishing a new SQL connection
func setupDatabaseTLS() error {
	// Register EdgelessDB root certificate
	log.Println("Attempting to connect to database...")
	pem := []byte(os.Getenv(marble.MarbleEnvironmentRootCA))
	rootCertPool := x509.NewCertPool()
	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		return errors.New("could not add root cert from PEM")
	}

	// Get certificate + private key for reader instance
	cert, err := tls.X509KeyPair([]byte(os.Getenv("CERT")), []byte(os.Getenv("KEY")))
	if err != nil {
		return err
	}

	mysql.RegisterTLSConfig("edgelessdb", &tls.Config{
		RootCAs:      rootCertPool,
		Certificates: []tls.Certificate{cert},
	})

	return nil
}
