# test locally
```bash
go run main.go
```

# Build the Docker image
```bash
docker build -t go-ref-lights .
```

# Run the Docker container
```bash
docker run -p 8080:8080 go-ref-lights
```

# ngrok instructions