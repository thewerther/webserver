package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/thewerther/webserver/internal/database"
)

type apiConfig struct {
	FileServerHits atomic.Int32
	database       *database.Queries
	JWT_Secret     string
  isAdmin        bool
}

func main() {
	const port string = "8080"
	const rootPath string = "."

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file", err)
	}
  dbURL := os.Getenv("DB_URL")
  if dbURL == "" {
    log.Fatal("DB_URL not provided in .env")
  }

	dbConn, err := sql.Open("postgres", dbURL)
	if err != nil {
    log.Fatalf("Error opening database: %s", err)
	}
  dbQueries := database.New(dbConn)

  isAdmin := os.Getenv("PLATFORM")
  if isAdmin == "" {
    log.Fatal("PLATFORM has to be set in .env")
  }

	apiCfg := &apiConfig{
    FileServerHits: atomic.Int32{},
    database: dbQueries,
    JWT_Secret: os.Getenv("JWT_SECRET"),
    isAdmin: isAdmin == "dev",
  }

	serveMux := http.NewServeMux()
	fileServerHandler := http.StripPrefix("/app", http.FileServer(http.Dir(rootPath)))
	serveMux.Handle("/app/", apiCfg.middlewareMetricsInc(fileServerHandler))

	serveMux.HandleFunc("GET /api/healthz", serveHealthz)

	serveMux.HandleFunc("POST /api/chirps", apiCfg.createChirp)
	serveMux.HandleFunc("GET /api/chirps", apiCfg.getChirps)
	serveMux.HandleFunc("GET /api/chirps/{chirpId}", apiCfg.getChirpByID)

	serveMux.HandleFunc("POST /api/users", apiCfg.createUser)
	//serveMux.HandleFunc("PUT /api/users", apiCfg.updateUser)
	serveMux.HandleFunc("POST /api/login", apiCfg.loginUser)

	serveMux.HandleFunc("GET /admin/metrics", apiCfg.serveAdminMetrics)
	serveMux.HandleFunc("POST /admin/reset", apiCfg.resetServer)

	server := &http.Server{Handler: serveMux, Addr: ":" + port}
	log.Printf("Serving on port: %s\n", port)
	log.Fatal(server.ListenAndServe())
}
