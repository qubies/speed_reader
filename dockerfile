from golang
RUN mkdir /go/src/speed_reader
workdir /go/src/speed_reader
COPY ./ /go/src/speed_reader
RUN go get -v && go build 
CMD ./speed_reader
