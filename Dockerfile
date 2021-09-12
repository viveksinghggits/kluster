FROM alpine

COPY kluster /usr/local/bin

ENTRYPOINT ["kluster"]