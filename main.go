package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
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
fileserverHits 	atomic.Int32
db      		*database.Queries
platform    	string
secret  		string
polkaKey        string
}

type userPerams struct {
	Password 	string  `json:"password"`
	Email    	string  `json:"email"`
}

type makeChirpParams struct {
	Body    	string  `json:"body"`
}

type chirpResponse struct {
	ID        uuid.UUID		`json:"id"`
	CreatedAt time.Time		`json:"created_at"`
	UpdatedAt time.Time		`json:"updated_at"`
	Body      string		`json:"body"`
	UserID    uuid.UUID		`json:"user_id"`
}

type isChirpRedWebhookRequest struct {
	Event string `json:"event"`
	Data  struct {
		UserID string `json:"user_id"`
	} `json:"data"`
}

type userInfoResponse struct {
	ID        	uuid.UUID `json:"id"`
	CreatedAt 	time.Time `json:"created_at"`
	UpdatedAt 	time.Time `json:"updated_at"`
	Email     	string    `json:"email"`
	IsChirpRed	bool	  `json:"is_chirpy_red"`
}

type userSessionResponse struct {
	ID        		uuid.UUID `json:"id"`
	CreatedAt 		time.Time `json:"created_at"`
	UpdatedAt 		time.Time `json:"updated_at"`
	Email     		string    `json:"email"`
	Token	  		string	  `json:"token"`
	RefreshToken 	string    `json:"refresh_token"`
	IsChirpRed		bool	  `json:"is_chirpy_red"`
}

type refreshTokenResponse struct {
	Token 	string  `json:"token"`
}

type errResponse struct {
	Error   string  `json:"error"`
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

	jwtSecret := os.Getenv("SECRET")

	if jwtSecret == "" {
		log.Fatal("JWT_SECRET must be set")
	}

	polka := os.Getenv("POLKA_KEY")

	if jwtSecret == "" {
		log.Fatal("POLKA_KEY must be set")
	}


	dbQueries := database.New(db)

	var apiConfig apiConfig

	apiConfig.db = dbQueries
	apiConfig.platform = platform
	apiConfig.secret = jwtSecret
	apiConfig.polkaKey = polka

	mux := http.NewServeMux()
	api := http.NewServeMux()
	admin := http.NewServeMux()

	admin.HandleFunc("GET /metrics", apiConfig.metricsHandler)
	admin.HandleFunc("POST /reset", apiConfig.resetHandler)
	api.HandleFunc("GET /healthz", healthzHandler)
	api.HandleFunc("POST /users", apiConfig.makeUserHandler)
	api.HandleFunc("PUT /users", apiConfig.updateUserHandler)
	api.HandleFunc("POST /login", apiConfig.loginHandler)
	api.HandleFunc("POST /refresh", apiConfig.tokenRefreshHandler)
	api.HandleFunc("POST /revoke", apiConfig.tokenRevokeHandler)
	api.HandleFunc("POST /chirps", apiConfig.chirpHandler)
	api.HandleFunc("GET /chirps", apiConfig.allChirpsHandler)
	api.HandleFunc("GET /chirps/{chirpID}", apiConfig.getChirpHandler)
	api.HandleFunc("DELETE /chirps/{chirpID}", apiConfig.deleteChirpHandler)
	api.HandleFunc("POST /polka/webhooks", apiConfig.isChirpRedWebhooksHandler)




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
	//expecting session/JWT token as bearer token
	decoder := json.NewDecoder(r.Body)

	var request makeChirpParams

	err := decoder.Decode(&request)

	if err != nil || len(request.Body) > 140 {
		statusCode := http.StatusInternalServerError

		errString :=fmt.Sprintf("Error, unable to create user: %v", err)

		if len(request.Body) > 140 {
			errString = fmt.Sprintf("Chirp is longer than 140 characters: %v", err)
			statusCode = http.StatusBadRequest
		}

		res := errResponse{
			Error: errString,
		}

		err = marshalHelper(w ,res, statusCode)
		if err != nil {
			fmt.Printf("create chirp: %v", err)
		}
		return
	}

	bearerToken, err := auth.GetBearerToken(r.Header)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		log.Printf("Unable to retrieve Bearer token: %v", err)
		return
	}

	userID, err := auth.ValidateJWT(bearerToken, cfg.secret)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		log.Printf("Failed to validate user: %v", err)
		return
	}

	// ----------- add profanity clean up here if needed/wanted -----------
	// replaceArr := []string{"kerfuffle", "sharbert", "fornax"}
	// content := strings.Split(request.Body, " ")
	// for i := range content {
	// 	for _, str := range replaceArr {
	// 		if str == strings.ToLower(content[i]) {
	// 			content[i] = strings.ToLower(content[i])
	// 			content[i] = strings.Replace(content[i], str, "****", -1)
	// 		}
	// 	}
	// }
	// request.Body = strings.Join(content, " ")

	var curChirp database.Chirp

	curChirp, err = cfg.db.CreateChirp(r.Context(), database.CreateChirpParams{
		Body: request.Body,
		UserID: userID,
	})

	if err != nil {
		log.Printf("Failed to create chirp")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	res := chirpResponse{
		ID: curChirp.ID,
		CreatedAt: curChirp.CreatedAt,
		UpdatedAt: curChirp.UpdatedAt,
		Body: curChirp.Body,
		UserID: curChirp.UserID,
	}

	err = marshalHelper(w ,res, http.StatusCreated)
	if err != nil {
		fmt.Printf("create chirp: %v", err)
	}
}


func (cfg *apiConfig) allChirpsHandler (w http.ResponseWriter, r *http.Request) {

	queryParams := r.URL.Query()

	// Returns the first value associated with "author_id"
	authorId := queryParams.Get("author_id")

	userID, err := uuid.Parse(authorId)

	if err != nil{
	log.Printf("Failed to parse author id chirps")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	allChirps, err := cfg.db.GetAllChirps(r.Context(), userID)

	if err != nil {
		log.Printf("Failed to retreive chirps")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var res []chirpResponse;

	for _, chirp := range allChirps {
		res = append(res, chirpResponse{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		})
	}

	err = marshalHelper(w ,res, http.StatusOK)
	if err != nil {
		fmt.Printf("get all chirps: %v", err)
	}
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

	res := chirpResponse{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID,
		}


		err = marshalHelper(w ,res, http.StatusOK)
		if err != nil {
		fmt.Printf("get chirp: %v", err)
	}
}


func (cfg *apiConfig) deleteChirpHandler (w http.ResponseWriter, r *http.Request) {
	//expecting session/JWT token as bearer token
	chirpID, err := uuid.Parse(r.PathValue("chirpID"))

	if err != nil {
		log.Printf("Failed to parse chirp ID")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	bearerToken, err := auth.GetBearerToken(r.Header)

	if err != nil {
		log.Printf("missing bearer token: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	userID, err := auth.ValidateJWT(bearerToken, cfg.secret)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		log.Printf("Failed to validate user: %v", err)
		return
	}

	chirp, err := cfg.db.GetChirp(r.Context(), chirpID)

	if err != nil {
		log.Printf("no chirp found: %v", err)
    	w.WriteHeader(http.StatusNotFound)
		return
	}

	if chirp.UserID != userID {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err = cfg.db.DeleteChirp(r.Context(), database.DeleteChirpParams{
		ID: chirp.ID,
		UserID: userID,
	})

	if err != nil {
		log.Printf("Failed to delete chirp: %v", err)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}


func (cfg *apiConfig) makeUserHandler (w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)

	var request userPerams

	err := decoder.Decode(&request)

	//Fail decoding request, return error message
	if err != nil {

		errString := fmt.Sprintf("Error decoding parameters, unable to create user: %v", err)

		res := errResponse{
			Error: errString,
		}

		err = marshalHelper(w ,res, http.StatusInternalServerError)
		if err != nil {
			fmt.Printf("create user: %v", err)
		}
		return
	}


	hashedPass, err := auth.HashPassword(request.Password)

	if err != nil {
		log.Printf("Password hash error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	user, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{Email: request.Email, HashedPassword: hashedPass})


	//Fail to add user request, return error message
	if err != nil {
		errString :=fmt.Sprintf("Error, unable to create user: %v", err)

		res := errResponse{
			Error: errString,
		}

		err = marshalHelper(w ,res, http.StatusInternalServerError)
		if err != nil {
			fmt.Printf("create user: %v", err)
		}
		return
	}



	res := userInfoResponse{
		ID: user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email: user.Email,
		IsChirpRed: user.IsChirpRed,
	}


	err = marshalHelper(w ,res, http.StatusCreated)
	if err != nil {
		fmt.Printf("create user: %v", err)
	}

}


func (cfg *apiConfig) loginHandler (w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)

	var request userPerams

	err := decoder.Decode(&request)

	//Fail decoding request, return error message
	if err != nil {

		errString := fmt.Sprintf("Error decoding parameters, unable to login: %v", err)

		res := errResponse{
			Error: errString,
		}

		err = marshalHelper(w ,res, http.StatusInternalServerError)
		if err != nil {
			fmt.Printf("user login: %v", err)
		}
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

	token, err := auth.MakeJWT(user.ID, cfg.secret, time.Hour)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		log.Printf("Unable to generate JWT: %v", err)
		return
	}

	refreshToken, err := auth.MakeRefreshToken()

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		log.Printf("Unable to generate refresh token: %v", err)
		return
	}

	err = cfg.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token: refreshToken,
		UserID: user.ID,
		ExpiresAt: time.Now().Add(time.Hour * 24 * 60),
	})

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Unable to add refresh token to DB: %v", err)
		return
	}

	res := userSessionResponse{
		ID: user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email: user.Email,
		Token: token,
		RefreshToken: refreshToken,
		IsChirpRed: user.IsChirpRed,
	}


	err = marshalHelper(w ,res, http.StatusOK)
	if err != nil {
		fmt.Printf("user login: %v", err)
	}

}


func (cfg *apiConfig) updateUserHandler (w http.ResponseWriter, r *http.Request) {
	//expecting session/JWT token as bearer token
	decoder := json.NewDecoder(r.Body)

	var request userPerams

	err := decoder.Decode(&request)

	//Fail decoding request, return error message
	if err != nil {

		errString := fmt.Sprintf("Error decoding parameters, unable to update user: %v", err)

		res := errResponse{
			Error: errString,
		}

		err = marshalHelper(w ,res, http.StatusInternalServerError)
		if err != nil {
			fmt.Printf("update user: %v", err)
		}
		return
	}

	bearerToken, err := auth.GetBearerToken(r.Header)

	if err != nil {
		err := marshalHelper(w ,bearerToken, http.StatusUnauthorized)
		if err != nil {
			fmt.Printf("update user: %v", err)
		}
		return
	}

	userID, err := auth.ValidateJWT(bearerToken, cfg.secret)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		log.Printf("Failed to validate user: %v", err)
		return
	}

	hashedPass, err := auth.HashPassword(request.Password)

	if err != nil {
		log.Printf("Password hash error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = cfg.db.UpdateUser(r.Context(), database.UpdateUserParams{ID: userID, Email: request.Email, HashedPassword: hashedPass})


	//Fail to add user request, return error message
	if err != nil {
		errString :=fmt.Sprintf("Error, unable to update user: %v", err)

		res := errResponse{
			Error: errString,
		}

		err = marshalHelper(w ,res, http.StatusInternalServerError)
		if err != nil {
			fmt.Printf("update user: %v", err)
		}
		return
	}

	user, err := cfg.db.GetUser(r.Context(),request.Email)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Incorrect email or password"})
		return
	}

	res := userInfoResponse{
		ID: user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email: user.Email,
		IsChirpRed: user.IsChirpRed,
	}


	err = marshalHelper(w ,res, http.StatusOK)
	if err != nil {
		fmt.Printf("create user: %v", err)
	}


}


func (cfg *apiConfig) isChirpRedWebhooksHandler (w http.ResponseWriter, r *http.Request) {
	//expects APIKey to be passed in header as Authorization: ApiKey <key>

	apiKey, err := auth.GetAPIKey(r.Header)

	if err != nil || apiKey != cfg.polkaKey {
		log.Printf("Rejected request, invalid APIKey")

		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	Decoder := json.NewDecoder(r.Body)

	var request isChirpRedWebhookRequest

	err = Decoder.Decode(&request)

	if err != nil {
		log.Printf("Failed to decode request: %v", err)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if request.Event != "user.upgraded" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	userID, err := uuid.Parse(request.Data.UserID)

	if err != nil {
		log.Printf("Failed parse user id: %v", err)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = cfg.db.UpgradeChirpRed(r.Context(), userID)

	if errors.Is(err, sql.ErrNoRows) {
		log.Printf("no rows found: %v", err)

		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err != nil {
		log.Printf("error upgrading user: %v", err)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}


	w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) tokenRefreshHandler (w http.ResponseWriter, r *http.Request) {
	//expecting refresh token as bearer token
	bearerToken, err := auth.GetBearerToken(r.Header)

	if err != nil {
		err := marshalHelper(w ,bearerToken, http.StatusUnauthorized)
		if err != nil {
			fmt.Printf("token refresh: %v", err)
		}
		return
	}

	refreshToken, err := cfg.db.IsValidRefreshToken(r.Context(), bearerToken)

	if err != nil {
		err := marshalHelper(w ,refreshToken, http.StatusUnauthorized)
		if err != nil {
			fmt.Printf("token refresh: %v", err)
		}
		return
	}

	newToken, err := auth.MakeJWT(refreshToken.UserID, cfg.secret, time.Hour)

	if err != nil {
		err := marshalHelper(w ,newToken, http.StatusUnauthorized)
		if err != nil {
			fmt.Printf("token refresh: %v", err)
		}
		return
	}

	tokenRefresh := refreshTokenResponse{
		Token: newToken,
	}

	err = marshalHelper(w ,tokenRefresh, http.StatusOK)

	if err != nil {
		fmt.Printf("token refresh: %v", err)
	}


}


func (cfg *apiConfig) tokenRevokeHandler (w http.ResponseWriter, r *http.Request) {
	//expecting refresh token as bearer token
	token, err := auth.GetBearerToken(r.Header)

	if err != nil {
		err := marshalHelper(w ,token, http.StatusUnauthorized)
		if err != nil {
			fmt.Printf("revoke token: %v", err)
		}
		return
	}

	err = cfg.db.RevokeToken(r.Context(), token)

	if err != nil {
		err := marshalHelper(w ,token, http.StatusUnauthorized)
		if err != nil {
			fmt.Printf("revoke token: %v", err)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}


func marshalHelper (w http.ResponseWriter, res any, statusCode int) error {
	data, err := json.Marshal(res)

	if err != nil {
		log.Printf("Error marshaling data")
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}


	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(data)
	return nil
}
