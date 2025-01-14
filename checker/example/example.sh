#!/bin/bash

# URL of the server to check
URL="http://$1:$2"

# Make a GET request to the URL and store the response body
response=$(curl --connect-timeout 5 -s "$URL")

# Check if the response contains "README"
if [[ "$response" == *"README"* ]]; then
    echo "UP"
else
    echo "DOWN"
fi
