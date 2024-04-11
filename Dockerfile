FROM golang:1.20-alpine as base

WORKDIR /app

COPY ./go.mod ./
COPY ./go.sum ./

# Download dependencies
RUN \
    go version && \
    go mod download

COPY ./ .

RUN go mod download

# RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /app/bin/open-fire


# ENV PORT=80

# EXPOSE 80
RUN chmod +x /app/build.sh

CMD [ "sh", "/app/build.sh" ]

# sudo docker build -t open-fire:0.0.1 -f ./open-fire/Dockerfile .
# sudo docker run --name open-fire-1 -p 80:80 -v .:/app/bin open-fire:0.0.1
