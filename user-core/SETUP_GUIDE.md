# Deployment Setup Guide

This guide provides clear instructions for deploying the `user-core` service using Docker and Heroku.

---

## Docker Deployment (Professional Setup)

### Prerequisites
- Install [Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/install/).
- Make sure you have PowerShell available to run scripts.

### Steps
1. **Clone Repository**:
   ```bash
   git clone <repository-url>
   cd user-core
   ```

2. **Build and Run Docker Container**:
   - Build the Image:
     ```bash
     docker build -t user-core .
     ```
   - Run the Container:
     ```bash
     docker run -p 8080:8080 user-core
     ```

3. **Verify Service**:
   - Access the service at `http://localhost:8080`.

4. **Optional (Compose for Multi-Service Environments)**:
   - Use `docker-compose.yml` (if added for dependent services).

### Notes
- The build process includes protobuf generation via `generate-proto.ps1`.

---

## Heroku Deployment (Student-Friendly Setup)

### Prerequisites
- Install the [Heroku CLI](https://devcenter.heroku.com/articles/heroku-cli).

### Steps
1. **Clone Repository**:
   ```bash
   git clone <repository-url>
   cd user-core
   ```

2. **Initialize Heroku App**:
   - Log in to Heroku:
     ```bash
     heroku login
     ```
   - Create a Heroku App:
     ```bash
     heroku create
     ```

3. **Deploy to Heroku**:
   - Use the `heroku.yml` definition for Docker-based builds.
     ```bash
     git add .
     git commit -m "Add Heroku support"
     git push heroku main
     ```

4. **Access Service**:
   - Visit the deployed URL provided in the output.

---

### Troubleshooting
If protobuf generation fails, verify:
- `buf` is installed: `https://buf.build`.
- Script `generate-proto.ps1` is executable with PowerShell.

---

## Envoy-Protected Boundary Setup

This repository now includes a top-level `docker-compose.yml` that runs `user-core` behind Envoy.

### Required Environment Variables

Set these before starting the stack:

- `ZITADEL_ISSUER` - your Zitadel OIDC issuer URL
- `ZITADEL_AUDIENCE` - the audience configured for the access token
- `ZITADEL_JWKS_URI` - the full JWKS endpoint URL used by Envoy
- `ZITADEL_JWKS_HOST` - hostname for the JWKS endpoint
- `ZITADEL_JWKS_PORT` - JWKS port, usually `443`

Optional local-only defaults still apply for `user-core` itself:

- `JWT_SECRET`
- `JWT_ISSUER`
- `JWT_ACCESS_TTL`
- `JWT_REFRESH_TTL`

### Start the Stack

```bash
docker compose up --build
```

### Test the Boundary

1. Get a valid access token from Zitadel for the configured audience.
2. Call the service through Envoy with the bearer token:

```bash
grpcurl -plaintext \
   -H "authorization: Bearer $TOKEN" \
   localhost:8080 \
   usercore.v1.UserService/ListUsers
```

3. Retry the same call without the token. Envoy should reject it before the request reaches `user-core`.
4. Inspect Envoy logs to confirm the JWT filter accepted the request and forwarded trusted identity headers such as `x-user-id` and `x-user-email`.

### Notes

- `user-core` is started with `--data-dir=/data` and `--config-dir=/config` from the compose file.
- The bootstrap seed file is mounted from `backend/config/bootstrap_users.example.textproto` as `bootstrap_users.textproto` inside the container.
- The shared middleware package is ready to read trusted caller identity from Envoy-injected metadata.