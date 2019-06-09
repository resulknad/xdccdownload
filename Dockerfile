FROM golang:alpine3.9

WORKDIR /go/src/app
COPY . .
RUN apk add git

RUN go get -d -v ./...
RUN go install -v ./...

EXPOSE 8082
VOLUME ["/root/", "/tvshows/", "/movies/"]
RUN ln -s /root/.config.json /.config.json 
#CMD ["/bin/sh"]
CMD ["app","-printLog","1"]
