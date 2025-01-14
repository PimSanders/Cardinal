# Cardinal Game Server

Cardinal is a Go-based game server designed to manage **Attack & Defend CTF competitions**. This guide outlines the project structure, setup process, and key features to help you deploy and manage the server.

---

## Table of Contents
1. [Project Structure](#project-structure)
2. [Setup Instructions](#setup-instructions)
   - [Prerequisites](#prerequisites)
   - [Building the Project](#building-the-project)
   - [Configuration](#configuration)
   - [Running the Application](#running-the-application)
   - [Stopping the Application](#stopping-the-application)
3. [Key Functionalities](#key-functionalities)
   - [Initialization](#initialization)
   - [API Endpoints](#api-endpoints)
   - [Flag Submission](#flag-submission)
4. [Docker Setup](#docker-setup)
   - [Docker Compose Services](#docker-compose-services)
      - [Cardinal](#cardinal-service)
      - [Database](#database-service)
      - [Web](#web-service)
5. [Docker Compose File](#docker-compose-file)

---

## Project Structure

```plaintext
./
├── gameserver
│   ├── cardinal             # Core game server code
│   ├── checker              # Checker scripts
│   ├── conf                 # Configuration files
│   ├── docker-compose.yml   # Docker Compose configuration
│   ├── README               # This file
│   └── web                  # Web service files (e.g., Nginx)
```

---

## Setup Instructions

### Prerequisites

- [Docker](https://www.docker.com/)
- [Docker Compose](https://docs.docker.com/compose/)

### Building the Project

1. Clone the repository:
    ```sh
    git clone https://github.com/BumbleB-NL/attack-defense.git
    cd gameserver
    ```

2. Build the Docker images:
    ```sh
    docker compose build
    ```

### Configuration

Edit the configuration file located at `conf/Cardinal.toml` to match your environment's requirements.

### Running the Application

Start the application with Docker Compose:
```sh
docker-compose up -d
```

This will launch the services defined in the Docker Compose file: **cardinal**, **db**, and **web**.

### Stopping the Application

Stop the application and remove the containers:
```sh
docker-compose down
```

---

## Key Functionalities

### Initialization

The initialization process (in `bootstrap.LinkStart`) performs the following tasks:
- Load configuration settings.
- Initialize the MySQL database.
- Set up the game timer.
- Initialize caches and webhooks.
- Start the web server.

### API Endpoints

API routes (defined in `route.Init`) include:
- `POST /api/flag` - Submit a flag.
- `POST /api/login` - Team login.
- `POST /api/logout` - Team logout.
- `GET /api/rank` - Fetch ranking data.
- `GET /api/asteroid` - WebSocket endpoint for asteroid interactions.

### Flag Submission

Flag submission logic (in `game.SubmitFlag`) includes:
- Verifying that the competition has started.
- Validating the authorization header.
- Checking the flag against current and future rounds.
- Recording the flag in the database.

---

## Docker Setup

### Overview

The server uses a multi-container setup with the following services:
1. **cardinal** - Core game server.
2. **db** - MySQL database.
3. **web** - Nginx reverse proxy.

Each service is defined in the `docker-compose.yml` file.

### Docker Compose Services

#### Cardinal Service

- **Build context:** `./cardinal`
- **Environment variable:** `CARDINAL_DOCKER=1`
- **Ports:** Exposes `19999` (game server port).
- **Volumes:** Mounts `./conf/Cardinal.toml` for configuration.
- **Logging:** Logs are capped at 200KB per file (max 10 files).
- **Restart policy:** Always restarts if stopped.
- **Dependencies:** Waits for the database service (`db`) to be ready.

#### Database Service

- **Image:** `mysql:8.0.21`
- **Ports:** Exposes `3306` (MySQL default port).
- **Volumes:** Persists data in `./Cardinal_database`.
- **Environment variables:** 
  - `MYSQL_ROOT_PASSWORD`: Root password.
  - `MYSQL_DATABASE`: Database name.
  - `MYSQL_USER`: User for the application.
  - `MYSQL_PASSWORD`: Password for the user.
- **Command:** Configured to use UTF-8 MB4 for better compatibility.
- **Logging:** Similar to the cardinal service.
- **Restart policy:** Always restarts if stopped.

#### Web Service

- **Build context:** `./web`
- **Ports:** Exposes `8087` on the host.
- **Volumes:** Mounts `./web/nginx.conf` for Nginx configuration.
- **Logging:** Configured with the same logging settings.
- **Restart policy:** Always restarts if stopped.

---

## Docker Compose File

Below is the full `docker-compose.yml` file for reference:

```yaml
services:
  cardinal:
    build:
      context: ./cardinal
    environment:
      CARDINAL_DOCKER: 1
    ports:
      - "19999:19999"
    volumes:
      - ./conf/Cardinal.toml:/Cardinal/conf/Cardinal.toml
      - ./checker/:/checker/
    logging:
      options:
        max-size: "200k"
        max-file: "10"
    restart: always
    depends_on:
      - db

  db:
    image: mysql:8.0.21
    ports:
      - "3306:3306"
    volumes:
      - ./Cardinal_database:/var/lib/mysql
    logging:
      options:
        max-size: "200k"
        max-file: "10"
    restart: always
    environment:
      MYSQL_ROOT_PASSWORD: Cardinal
      MYSQL_DATABASE: Cardinal
      MYSQL_USER: Cardinal
      MYSQL_PASSWORD: Cardinal
    command: mysqld --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci

  web:
    build:
      context: ./web
    ports:
      - 8087:80
    volumes:
      - ./web/nginx.conf:/etc/nginx/conf.d/default.conf
    logging:
      options:
        max-size: "200k"
        max-file: "10"
    restart: always
```

---

For detailed implementation and troubleshooting, explore the codebase and configuration files.