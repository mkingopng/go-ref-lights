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

## Todo:
- fix platform ready timer. its currently not working
- 'stop timer' button not working
- 'reset' not working
- green dots
- second timer for 'next attempt' triggered by 3 decisions