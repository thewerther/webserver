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

type ApiConfig struct {
	FileServerHits atomic.Int32
	Database       *database.Queries
	JWT_Secret     string
	IsAdmin        bool
	PolkaKey       string
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

	polkaKey := os.Getenv("POLKA_KEY")
	if polkaKey == "" {
		log.Fatal("POLKA_KEY is not set in .env")
	}

	apiCfg := &ApiConfig{
		FileServerHits: atomic.Int32{},
		Database:       dbQueries,
		JWT_Secret:     os.Getenv("JWT_SECRET"),
		IsAdmin:        isAdmin == "dev",
		PolkaKey:       polkaKey,
	}

	serveMux := http.NewServeMux()
	fileServerHandler := http.StripPrefix("/app", http.FileServer(http.Dir(rootPath)))
	serveMux.Handle("/app/", apiCfg.middlewareMetricsInc(fileServerHandler))

	serveMux.HandleFunc("GET /api/healthz", serveHealthz)

	serveMux.HandleFunc("POST /api/chirps", apiCfg.createChirp)
	serveMux.HandleFunc("GET /api/chirps", apiCfg.getChirps)
	serveMux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.getChirpByID)
	serveMux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.deleteChirpByID)

	serveMux.HandleFunc("POST /api/users", apiCfg.createUser)
  serveMux.HandleFunc("PUT /api/users", apiCfg.updateUser)
	serveMux.HandleFunc("POST /api/login", apiCfg.loginUser)
	serveMux.HandleFunc("POST /api/refresh", apiCfg.refreshToken)
	serveMux.HandleFunc("POST /api/revoke", apiCfg.revokeRefreshToken)

	serveMux.HandleFunc("GET /admin/metrics", apiCfg.serveAdminMetrics)
	serveMux.HandleFunc("POST /admin/reset", apiCfg.resetServer)

	serveMux.HandleFunc("POST /api/polka/webhooks", apiCfg.paymentHandler)

	server := &http.Server{Handler: serveMux, Addr: ":" + port}
	log.Printf("Serving on port: %s\n", port)
	log.Fatal(server.ListenAndServe())
}
