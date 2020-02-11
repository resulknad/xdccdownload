FROM golang:1-alpine

WORKDIR /go/src/app
COPY . .
RUN apk add git

RUN go get -d -v ./...
RUN go install -v ./...

EXPOSE 8082
VOLUME ["/root/", "/tvshows/", "/movies/"]
RUN ln -s /root/.config.json /.config.json 
# sudo docker run -it xdccdownload_downloader /bin/as
#CMD ["/bin/sh"]
CMD ["/go/bin/xdccdownload","-printLog","1"]
