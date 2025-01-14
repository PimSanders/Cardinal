# This Dockerfile builds and serves a multi-stage Node.js application using Nginx.
# 
# Stage 1: cardinal_frontend
# - Uses Node.js 14 LTS version.
# - Sets the DEBIAN_FRONTEND to noninteractive to avoid prompts during package installations.
# - Updates and installs necessary dependencies including build-essential, python3, and libgomp1.
# - Sets the working directory to /app.
# - Copies the frontend package.json and package-lock.json files to the working directory.
# - Installs frontend dependencies using yarn with strict SSL disabled and frozen lockfile.
# - Copies the frontend application code to the working directory.
# - Lints and builds the frontend application.
# 
# Stage 2: cardinal_manager
# - Uses Node.js 16 LTS version.
# - Sets the DEBIAN_FRONTEND to noninteractive to avoid prompts during package installations.
# - Updates and installs necessary dependencies including build-essential, python3, and libgomp1.
# - Sets the working directory to /app.
# - Copies the manager package.json and package-lock.json files to the working directory.
# - Installs manager dependencies using yarn with strict SSL disabled and frozen lockfile.
# - Copies the manager application code to the working directory.
# - Lints and builds the manager application.
# 
# Stage 3: Final image
# - Uses the Nginx base image.
# - Copies the built frontend files from the cardinal_frontend stage to the Nginx html directory.
# - Copies the built manager files from the cardinal_manager stage to the Nginx html/manager directory.
# Use the latest stable Node.js 14 LTS version
FROM node:14 AS cardinal_frontend

# Avoid interactive prompts during package installations
ARG DEBIAN_FRONTEND=noninteractive

# Update and install dependencies
RUN apt-get update -qq && \
    apt-get install -qq -y --no-install-recommends build-essential python3 libgomp1 && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Set the working directory
WORKDIR /app

# Copy and install dependencies
COPY ./frontend/package*.json ./
RUN yarn config set strict-ssl false && yarn install --frozen-lockfile

# Copy the application code
COPY ./frontend .

# Lint and build the application
RUN yarn lint && yarn build

# Use the latest stable Node.js 16 LTS version
FROM node:16 AS cardinal_manager

# Avoid interactive prompts during package installations
ARG DEBIAN_FRONTEND=noninteractive

# Update and install dependencies
RUN apt-get update -qq && \
    apt-get install -qq -y --no-install-recommends build-essential python3 libgomp1 && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

# Set the working directory
WORKDIR /app

# Copy and install dependencies
COPY ./manager/package*.json ./
RUN yarn config set strict-ssl false && yarn install --frozen-lockfile

# Copy the application code
COPY ./manager .

# Lint and build the application
RUN yarn lint && yarn build

# Copy the built files to the final image
FROM nginx
COPY --from=cardinal_frontend /app/dist/files /usr/share/nginx/html
COPY --from=cardinal_manager /app/dist/files/manager /usr/share/nginx/html/manager
COPY ./scoreboard/scoreboard.html /usr/share/nginx/html/scoreboard/index.html