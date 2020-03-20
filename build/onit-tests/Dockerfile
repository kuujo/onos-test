FROM alpine:3.8

RUN apk upgrade --update --no-cache && apk add git

RUN addgroup -S test && adduser -S test -G test

USER test

ADD _output/bin/onit-tests /usr/local/bin/onit-tests

ENTRYPOINT ["onit-tests"]