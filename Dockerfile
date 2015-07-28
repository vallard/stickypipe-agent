FROM golang:onbuild
MAINTAINER vallard@benincosa.com
WORKDIR /go/src/app
COPY . /go/src/app
RUN go build -o sp-agent main.go
ENTRYPOINT ["/go/src/app/sp-agent"]

