# The base go-image
FROM golang:1.15-alpine
 
# Create a directory for the app
RUN mkdir /app
 
WORKDIR /app
COPY go.mod /app/go.mod
COPY go.sum /app/go.sum

RUN go mod download 

# Copy all files from the current directory to the app directory
COPY cmds/server/server.go /app/server.go
 
# Run command as described:
# go build will build an executable file named server in the current directory
RUN go build -o server . 
 
# Run the server executable
CMD [ "/app/server" ]