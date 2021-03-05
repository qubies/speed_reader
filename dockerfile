from golang
RUN mkdir /go/src/speed_reader
workdir /go/src/speed_reader
COPY ./ /go/src/speed_reader
RUN go get -v 
RUN go build .
CMD go build . && ./speed_reader
