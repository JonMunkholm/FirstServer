# Server

## Description

### How it was coded

1. [Golang](https://go.dev/)
1. [Postgresql](https://www.postgresql.org/) - local DB
1. [SQLC](https://sqlc.dev/) - automated code gen for generating DB querying functions from SQL
1. [Goose](https://github.com/pressly/goose) - used as a DB migration tool

### What does it do

Is the backend for a chat service, which allows for messages up to 140 characters.

## End points

### Main page - place holder

1. Place holder for front end, displays **"Welcome to Chirpy"**.

`localhoast:<portNum>/app`

### Admin - for development or tracking

1. (GET Method) Visiting this URL will display the number of times the main app page has been visited (ie. the number of server Hits for the main page). Also available as an end point and will return status ok if successfuly queried.

`localhoast:<portNum>/admin/metrics`

2. (POST Method) Reset endpoint to reset DB tables - useful for testing.

`localhoast:<portNum>/admin/reset`

### API End Points - the meat and potatos of the server

1. (POST Method) Create a user - hashes password and stores in DB for login validation later.

`localhoast:<portNum>/api/users`

    1. expected request body format

    `{
        password: <1234SomePassword>
        email: <email@something.com>
    }`

    1. response body

    if request was good (201)

    `{
        id: <UserId>
        created_at: <Time>
        updated_at: <Time>
        email: <email>
        is_chirpy_red: <bool>
    }`

    if request was bad (4XX)

    `{
        error: <error>
    }`

2. (POST Method) Login request - validates, user email and password against email and hashed password in DB. Generates and returns session, refresh token(s) and if the account is "chirp red".

   `localhoast:<portNum>/api/login`

   1. expected request body format

   `{
    password: <1234SomePassword>
    email: <email@something.com>
}`

   1. response body

   if request was good (201)

   `{
    id: <UserId>
    created_at: <Time>
    updated_at: <Time>
    email: <email>
    token: <sessionToken>
    refresh_token: <refreshToken>
    is_chirpy_red: <bool>
}`

   if request was bad (4XX)

   `{
    error: <error>
}`

3. (PUT Method) Update user request - validates, session token and password against hashed password in DB. Retreives user_id from DB, uses this user_id to update the user email/password in the DB and returns an updated user object.

   `localhoast:<portNum>/api/users`

   1. expected request body format

   `{
    password: <1234SomePassword>
    email: <email@something.com>
}`

**AND**

    Request Header

`Authorization: Bearer <sessionToken/JWT>`

1.  response body

if request was good (201)

`{
    id: <UserId>
    created_at: <Time>
    updated_at: <Time>
    email: <email(updated)>
    is_chirpy_red: <bool>
}`

if request was bad (4XX)

`{
    error: <error>
}`

<!-- api.HandleFunc("POST /refresh", apiConfig.tokenRefreshHandler)
api.HandleFunc("POST /revoke", apiConfig.tokenRevokeHandler)
api.HandleFunc("POST /chirps", apiConfig.chirpHandler)
api.HandleFunc("GET /chirps", apiConfig.allChirpsHandler)
api.HandleFunc("GET /chirps/{chirpID}", apiConfig.getChirpHandler)
api.HandleFunc("DELETE /chirps/{chirpID}", apiConfig.deleteChirpHandler)
api.HandleFunc("POST /polka/webhooks", apiConfig.isChirpRedWebhooksHandler) -->
