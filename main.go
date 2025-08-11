package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/JonMunkholm/server/internal/auth"
	"github.com/JonMunkholm/server/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

const filepathRoot = "."
const port = "8080"


type apiConfig struct {
fileserverHits atomic.Int32
db      *database.Queries
platform    string
}


func main() {

	godotenv.Load()

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL must be set")
	}
	platform := os.Getenv("PLATFORM")
	if platform == "" {
		log.Fatal("PLATFORM must be set")
	}

	db, err := sql.Open("postgres", dbURL)

	if err != nil {
		log.Fatal("Failed to connect to DB", err)
	}

	dbQueries := database.New(db)

	mux := http.NewServeMux()
	api := http.NewServeMux()
	admin := http.NewServeMux()
	var apiConfig apiConfig

	apiConfig.db = dbQueries
	apiConfig.platform = platform

	admin.HandleFunc("GET /metrics", apiConfig.metricsHandler)
	admin.HandleFunc("POST /reset", apiConfig.resetHandler)
	api.HandleFunc("GET /healthz", healthzHandler)
	api.HandleFunc("POST /users", apiConfig.makeUserHandler)
	api.HandleFunc("POST /login", apiConfig.loginHandler)
	api.HandleFunc("POST /chirps", apiConfig.chirpHandler)
	api.HandleFunc("GET /chirps", apiConfig.allChirpsHandler)
	api.HandleFunc("GET /chirps/{chirpID}", apiConfig.getChirpHandler)



	fileServer := http.FileServer(http.Dir(filepathRoot))

	mux.Handle("/app/", apiConfig.middlewareMetricsInc(http.StripPrefix("/app", fileServer)))
	mux.Handle("/api/", http.StripPrefix("/api", api))
	mux.Handle("/admin/", http.StripPrefix("/admin", admin))




	server := &http.Server{

		Handler: mux,

		Addr:    ":" + port,

		// ReadTimeout:  10 * time.Second,
		// WriteTimeout: 10 * time.Second,
		// IdleTimeout:  30 * time.Second,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(server.ListenAndServe())
}

func healthzHandler (w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("OK"))
	if err != nil {
		log.Printf("Error writing healthz response: %v", err)
	}
}

func (cfg *apiConfig) metricsHandler (w http.ResponseWriter, r *http.Request){
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	count := cfg.fileserverHits.Load()
	// fmt.Fprintf(w, "Hits: %d\n", count)
	content := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", count)
	w.Write([]byte(content))
}

func (cfg *apiConfig) resetHandler (w http.ResponseWriter, r *http.Request){

	if cfg.platform != "dev" {

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		return

	}

	err := cfg.db.ResetUsers(r.Context())

	//Fail to delete users, return error message
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to reset the database: " + err.Error()))
		return
	}

	err = cfg.db.ResetChirps(r.Context())
	//Fail to delete chirps, return error message
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to reset the database: " + err.Error()))
		return
	}

	cfg.fileserverHits.Store(0)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0 and database reset to initial state."))
}

func (cfg *apiConfig) middlewareMetricsInc (next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}


func (cfg *apiConfig) chirpHandler (w http.ResponseWriter, r *http.Request){
	statusCode := http.StatusCreated
	type param struct {
		Body    	string  	`json:"body"`
		UserID 		uuid.UUID   `json:"user_id"`
	}

	decoder := json.NewDecoder(r.Body)

	var request param
	err := decoder.Decode(&request)

	if err != nil || len(request.Body) > 140 {
		statusCode := http.StatusInternalServerError

		type response struct {
			Error   string  `json:"error"`
		}

		res := response{
			Error: "Unable to decode request body",
		}

		if len(request.Body) > 140 {
			res.Error = "Chirp is longer than 140 characters"
			statusCode = http.StatusBadRequest
		}
		data, err := json.Marshal(res)

		if err != nil {
			log.Printf("Error marshaling data")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write(data)
		return
	}


	replaceArr := []string{"kerfuffle", "sharbert", "fornax"}
	content := strings.Split(request.Body, " ")
	for i := range content {
		for _, str := range replaceArr {
			if str == strings.ToLower(content[i]) {
				content[i] = strings.ToLower(content[i])
				content[i] = strings.Replace(content[i], str, "****", -1)
			}
		}
	}
	request.Body = strings.Join(content, " ")

	var curChirp database.Chirp

	curChirp, err = cfg.db.CreateChirp(r.Context(), database.CreateChirpParams{
		Body: request.Body,
		UserID: request.UserID,
	})

	if err != nil {
		statusCode = http.StatusInternalServerError
		log.Printf("Failed to create chirp")
		w.WriteHeader(statusCode)
		return
	}

	type Chirp struct {
		ID        uuid.UUID		`json:"id"`
		CreatedAt time.Time		`json:"created_at"`
		UpdatedAt time.Time		`json:"updated_at"`
		Body      string		`json:"body"`
		UserID    uuid.UUID		`json:"user_id"`
	}

	res := Chirp {
		ID: curChirp.ID,
		CreatedAt: curChirp.CreatedAt,
		UpdatedAt: curChirp.UpdatedAt,
		Body: curChirp.Body,
		UserID: curChirp.UserID,
	}

	data, err := json.Marshal(res)

	if err != nil {
		statusCode = http.StatusInternalServerError
		log.Printf("Error marshaling data")
		w.WriteHeader(statusCode)
		return
	}


	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(data)
}



func (cfg *apiConfig) makeUserHandler (w http.ResponseWriter, r *http.Request) {

	type param struct {
		Password string  `json:"password"`
		Email    string  `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)

	var request param

	err := decoder.Decode(&request)

	//Fail decoding request, return error message
	if err != nil {

		type response struct {
			Error   string  `json:"error"`
		}

		res := response{
			Error: "Error decoding parameters, unable to create user",
		}

		data, err := json.Marshal(res)

		if err != nil {
			log.Printf("Error marshaling data")
			} else {
				log.Printf("Error decoding parameters: %s", err)
			}


		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(data)
		return
	}


	hashedPass, err := auth.HashPassword(request.Password)
	if err != nil {
		log.Printf("Password hash error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	usr, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{Email: request.Email, HashedPassword: hashedPass})


	//Fail to add user request, return error message
	if err != nil {
		fmt.Printf("something went wrong: %v", err)
		type response struct {
			Error   string  `json:"error"`
		}

		res := response{
			Error: "Error, unable to create user",
		}

		data, err := json.Marshal(res)

		if err != nil {
			log.Printf("Error marshaling data")
		}


		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(data)
		return
	}

	type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	}

	var user User

	user.ID = usr.ID
	user.CreatedAt = usr.CreatedAt
	user.UpdatedAt = usr.UpdatedAt
	user.Email = usr.Email

	data, err := json.Marshal(user)

		if err != nil {
			log.Printf("Error marshaling data")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}


		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(data)

}


func (cfg *apiConfig) allChirpsHandler (w http.ResponseWriter, r *http.Request) {

	allChirps, err := cfg.db.GetAllChirps(r.Context())

	if err != nil {
		log.Printf("Failed to retreive chirps")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	type ChirpResponse struct {
		ID        uuid.UUID		`json:"id"`
		CreatedAt time.Time		`json:"created_at"`
		UpdatedAt time.Time		`json:"updated_at"`
		Body      string		`json:"body"`
		UserID    uuid.UUID		`json:"user_id"`
	}

	var res []ChirpResponse;

	for _, chirp := range allChirps {
		res = append(res, ChirpResponse{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		})
	}

	data, err := json.Marshal(res)

	if err != nil {
		log.Printf("Error marshaling data")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}


	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}


func (cfg *apiConfig) getChirpHandler (w http.ResponseWriter, r *http.Request) {

	chirpID, err := uuid.Parse(r.PathValue("chirpID"))

	if err != nil {
		log.Printf("Failed to parse chirp ID")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	chirp, err := cfg.db.GetChirp(r.Context(), chirpID)

	if err != nil {
		log.Printf("Failed to retreive chirp")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	type ChirpResponse struct {
		ID        uuid.UUID		`json:"id"`
		CreatedAt time.Time		`json:"created_at"`
		UpdatedAt time.Time		`json:"updated_at"`
		Body      string		`json:"body"`
		UserID    uuid.UUID		`json:"user_id"`
	}

	res := ChirpResponse{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		}


	data, err := json.Marshal(res)

	if err != nil {
		log.Printf("Error marshaling data")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}


	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}



func (cfg *apiConfig) loginHandler (w http.ResponseWriter, r *http.Request) {
	type param struct {
		Password string  `json:"password"`
		Email    string  `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)

	var request param

	err := decoder.Decode(&request)

	//Fail decoding request, return error message
	if err != nil {

		type response struct {
			Error   string  `json:"error"`
		}

		res := response{
			Error: "Error decoding parameters, unable to create user",
		}

		data, err := json.Marshal(res)

		if err != nil {
			log.Printf("Error marshaling data")
			} else {
				log.Printf("Error decoding parameters: %s", err)
			}


		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(data)
		return
	}

	user, err := cfg.db.GetUser(r.Context(),request.Email)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Incorrect email or password"})
		return
	}

	err = auth.CheckPasswordHash(request.Password, user.HashedPassword)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Incorrect email or password"})
		return
	}

	type userResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	}

	var res userResponse

	res.ID = user.ID
	res.CreatedAt = user.CreatedAt
	res.UpdatedAt = user.UpdatedAt
	res.Email = user.Email

	data, err := json.Marshal(res)

		if err != nil {
			log.Printf("Error marshaling data")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}


		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(data)
}
