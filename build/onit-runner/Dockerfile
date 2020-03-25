FROM alpine:3.8

RUN apk upgrade --update --no-cache

RUN addgroup -S onit && adduser -S onit -G onit

USER onit

ADD _output/bin/onit-runner /usr/local/bin/onit-runner

WORKDIR /home/onit

ENTRYPOINT ["onit-runner"]