FROM golang:1.12
MAINTAINER auxten

RUN apt-get update && apt-get install -y --no-install-recommends \
		libicu-dev \
	&& rm -rf /var/lib/apt/lists/*


RUN go get github.com/mattn/goveralls
RUN go get github.com/haya14busa/goverage
RUN go get golang.org/x/lint/golint
RUN go get github.com/haya14busa/reviewdog/cmd/reviewdog

WORKDIR $GOPATH