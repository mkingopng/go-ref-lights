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



# the structure looks like this:

go-ref-lights/
├── controllers/
│   └── page_controller.go
├── middleware/
│   └── auth.go
├── services/
│   └── qrcode_service.go
├── websocket/
│   └── handler.go
├── static/
│   ├── css/
│   │   └── styles.css
│   └── js/
│       ├── websocket.js
│       ├── lights.js
│       ├── left.js
│       ├── centre.js
│       └── right.js
├── templates/
│       └── lights.html
├── main.go
├── go.mod
└── go.sum
