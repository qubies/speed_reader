from golang
RUN mkdir /go/src/speed_reader
workdir /go/src/speed_reader
cmd go get && go build && ./speed_reader
