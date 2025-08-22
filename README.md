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

2.  (POST Method) Login request - validates, user email and password against email and hashed password in DB. Generates and returns session, refresh token(s) and if the account is "chirp red".

        `localhoast:<portNum>/api/login`

        1. expected request body format

        `{
        password: <1234SomePassword>
        email: <email@something.com>

    }`

        1. response body

        if request was good (200)

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

3.  (PUT Method) Update user request - validates, session token and password against hashed password in DB. Retreives user_id from DB, uses this user_id to update the user email/password in the DB and returns an updated user object.

        `localhoast:<portNum>/api/users`

        1. expected request body format

        `{
        password: <1234SomePassword>
        email: <email@something.com>

    }`

        **AND**

        Request Header

                `Authorization: Bearer <sessionToken/JWT>`

        1. response body

        if request was good (200)

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

4.  (POST Method) Token Refresh request - validates the user refresh token and generates a new session/JWT token and returns the token in the response body.

        `localhoast:<portNum>/api/refresh`

        1. expected request header to contain the refresh token from loging in

           `Authorization: Bearer <RefreshToken>`

        1. response body

        if request was good (200)

        `{
        token: <sessionToken>

    }`

        if request was bad (4XX)

5.  (POST Method) Token Revoke request - Revokes the access of a refresh token, does not return a response body, but will return a header http.status response.

        `localhoast:<portNum>/api/revoke`

        1. expected request header to contain the refresh token from loging in

           `Authorization: Bearer <RefreshToken>`

        1. response body

        if request was good (204)


        if request was bad (4XX)

6.  (POST Method) Make Chirps request - Validates session/JWT token and uses token to retreive userID from DB. THen uses userID and requesxt body to create chirp and add to DB. Returns Chirp information in response.

        `localhoast:<portNum>/api/Chirps`

        1. expected request body format

        `{
        body: <140Chars>

    }`

        **AND**

        Request Header

                `Authorization: Bearer <sessionToken/JWT>`

        1. response body

        if request was good (201)

        `{
            id: <id>
            created_at: <created_at>
            updated_at: <updated_at>
            body: <body>
            user_id: <user_id>
        }`

        if request was bad (4XX)

        `{
        error: <error>

    }`

7.  (GET Method) Get all Chirps or all chirps attributed to a specific author if authorID is incluede in URL queries - Checks URL for query paramaters and gets the authorID if included. Then retreives all chirps with an optional filter including the authorID; if included filters for all chirps from that author, else returns all chirps in DB

        `localhoast:<portNum>/api/Chirps`

        1. optional query param in URL

        `localhoast:<portNum>/api/chirps?author_id=${authorID}`

        1. response body

        if request was good (200)

        `[{
        id: <id>
        created_at: <created_at>
        updated_at: <updated_at>
        body: <body>
        user_id: <user_id>

    }, {
    id: <id>
    created_at: <created_at>
    updated_at: <updated_at>
    body: <body>
    user_id: <user_id>
    }, ...]`

        if request was bad (4XX)

        `{
        error: <error>

}`

8.  (GET Method) Get a specific Chirp using the ChripID in the request - Retreives a specific chirp from the DB if the chirp exists.

        `localhoast:<portNum>/api/Chirps/{chirpID}`

        1. response body

        if request was good (200)

        `{
        id: <id>
        created_at: <created_at>
        updated_at: <updated_at>
        body: <body>
        user_id: <user_id>

    }`

        if request was bad (4XX)

        `{
        error: <error>

}`

9.  (DELETE Method) Deletes a specific Chirp using the ChripID in the request - Removes a specific chirp from the DB if the chirp exists.

    `localhoast:<portNum>/api/Chirps/{chirpID}`

    1. response body

    if request was good (204)

    if request was bad (4XX)

10. (POST Method) Hypothetical Webhook end point that flags an account as ChirpRed if they have paid via Polka (not a real payment processor). Validates the session API key against an API key from the request header and checks the request body for the userID. Depending on request body, the account will be marked as ChirpRed. The API keys must match else the request will fail.

`localhoast:<portNum>/api/polka/webhooks`

    1.  expected request body format

        `{
        event: <UpdateChirpRed>
        data: {
            user_id: <userID>
        }
    }`

    **AND**

    Request Header

                `Authorization: ApiKey <apiKey>`

    1. response body

        if request was good (204)

        if request was bad (4XX)
