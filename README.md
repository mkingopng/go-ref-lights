# Build the Docker image
docker build -t go-ref-lights .

# Run the Docker container
docker run -p 8080:8080 go-ref-lights

b874-167-179-162-245.ngrok-free.app
