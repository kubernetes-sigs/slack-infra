FROM alpine
RUN apk update && apk add tzdata ca-certificates
ADD tempelis /tempelis
ENTRYPOINT ["/tempelis"]
