# Server Documentation

## Overview

This server powers **Chirpy**, a chat service that allows messages up to 140 characters.

### Tech Stack

- [Golang](https://go.dev/)
- [PostgreSQL](https://www.postgresql.org/) — local DB
- [SQLC](https://sqlc.dev/) — generates DB query functions from SQL
- [Goose](https://github.com/pressly/goose) — DB migration tool

---

## Endpoints

### Main Page

**GET** `/app`
Displays a placeholder page with **"Welcome to Chirpy"**.

```bash
curl http://localhost:<port>/app
```

---

### Admin Endpoints

#### 1. Metrics

**GET** `/admin/metrics`
Returns the number of times the main app page has been visited.

**Response:**

```json
{
  "status": "ok",
  "hits": 42
}
```

```bash
curl http://localhost:<port>/admin/metrics
```

---

#### 2. Reset

**POST** `/admin/reset`
Resets all DB tables (useful for testing).

```bash
curl -X POST http://localhost:<port>/admin/reset
```

---

### API Endpoints

---

#### 1. Create User

**POST** `/api/users`
Creates a new user, hashes password, stores in DB.

**Request:**

```json
{
  "password": "1234SomePassword",
  "email": "email@something.com"
}
```

**Response (201):**

```json
{
  "id": "UserId",
  "created_at": "Time",
  "updated_at": "Time",
  "email": "email@something.com",
  "is_chirpy_red": false
}
```

```bash
curl -X POST http://localhost:<port>/api/users \
  -H "Content-Type: application/json" \
  -d '{"password": "1234SomePassword", "email": "email@something.com"}'
```

---

#### 2. Login

**POST** `/api/login`
Validates credentials and returns a session + refresh token.

**Request:**

```json
{
  "password": "1234SomePassword",
  "email": "email@something.com"
}
```

**Response (200):**

```json
{
  "id": "UserId",
  "created_at": "Time",
  "updated_at": "Time",
  "email": "email@something.com",
  "token": "sessionToken",
  "refresh_token": "refreshToken",
  "is_chirpy_red": false
}
```

```bash
curl -X POST http://localhost:<port>/api/login \
  -H "Content-Type: application/json" \
  -d '{"password": "1234SomePassword", "email": "email@something.com"}'
```

---

#### 3. Update User

**PUT** `/api/users`
Updates user email/password. Requires session token.

**Headers:**

```
Authorization: Bearer <sessionToken>
```

**Request:**

```json
{
  "password": "newPassword",
  "email": "new@email.com"
}
```

**Response (200):**

```json
{
  "id": "UserId",
  "created_at": "Time",
  "updated_at": "Time",
  "email": "new@email.com",
  "is_chirpy_red": false
}
```

```bash
curl -X PUT http://localhost:<port>/api/users \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <sessionToken>" \
  -d '{"password": "newPassword", "email": "new@email.com"}'
```

---

#### 4. Refresh Token

**POST** `/api/refresh`
Generates a new session token from a refresh token.

**Headers:**

```
Authorization: Bearer <refreshToken>
```

**Response (200):**

```json
{
  "token": "newSessionToken"
}
```

```bash
curl -X POST http://localhost:<port>/api/refresh \
  -H "Authorization: Bearer <refreshToken>"
```

---

#### 5. Revoke Token

**POST** `/api/revoke`
Revokes a refresh token. No body in response.

**Headers:**

```
Authorization: Bearer <refreshToken>
```

**Response:** `204 No Content`

```bash
curl -X POST http://localhost:<port>/api/revoke \
  -H "Authorization: Bearer <refreshToken>"
```

---

#### 6. Create Chirp

**POST** `/api/chirps`
Creates a new chirp (max 140 chars). Requires session token.

**Headers:**

```
Authorization: Bearer <sessionToken>
```

**Request:**

```json
{
  "body": "Hello Chirpy!"
}
```

**Response (201):**

```json
{
  "id": "id",
  "created_at": "Time",
  "updated_at": "Time",
  "body": "Hello Chirpy!",
  "user_id": "UserId"
}
```

```bash
curl -X POST http://localhost:<port>/api/chirps \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <sessionToken>" \
  -d '{"body": "Hello Chirpy!"}'
```

---

#### 7. Get Chirps

**GET** `/api/chirps`
Returns all chirps or filters by `author_id`.

**Query Param:**

```
/api/chirps?author_id=<authorID>
```

**Response (200):**

```json
[
  {
    "id": "id",
    "created_at": "Time",
    "updated_at": "Time",
    "body": "Hello Chirpy!",
    "user_id": "UserId"
  }
]
```

```bash
curl http://localhost:<port>/api/chirps
curl http://localhost:<port>/api/chirps?author_id=123
```

---

#### 8. Get Chirp by ID

**GET** `/api/chirps/{chirpID}`
Fetches a single chirp by ID.

**Response (200):**

```json
{
  "id": "id",
  "created_at": "Time",
  "updated_at": "Time",
  "body": "Hello Chirpy!",
  "user_id": "UserId"
}
```

```bash
curl http://localhost:<port>/api/chirps/123
```

---

#### 9. Delete Chirp

**DELETE** `/api/chirps/{chirpID}`
Deletes a chirp by ID.

**Response:** `204 No Content`

```bash
curl -X DELETE http://localhost:<port>/api/chirps/123
```

---

#### 10. Webhook (Polka)

**POST** `/api/polka/webhooks`
Flags a user as **ChirpyRed** after a (mock) Polka payment.

**Headers:**

```
Authorization: ApiKey <apiKey>
```

**Request:**

```json
{
  "event": "UpdateChirpRed",
  "data": {
    "user_id": "UserId"
  }
}
```

**Response:** `204 No Content`

```bash
curl -X POST http://localhost:<port>/api/polka/webhooks \
  -H "Content-Type: application/json" \
  -H "Authorization: ApiKey <apiKey>" \
  -d '{"event": "user.upgraded", "data": {"user_id": "123"}}'
```
