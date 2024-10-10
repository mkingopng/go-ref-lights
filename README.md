# test locally
```bash
go run main.go
```

# build the Docker image
```bash
docker build -t go-ref-lights .
```

# run the Docker container
```bash
docker run -p 8080:8080 go-ref-lights
```

## Todo:
- fix platform ready timer. its currently not working
- 'stop timer' button not working
- 'reset' not working
- green dots
- second timer for 'next attempt' triggered by 3 decisions

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
