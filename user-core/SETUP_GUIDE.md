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