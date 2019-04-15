FROM alpine
RUN apk update && apk add tzdata ca-certificates
ADD slack-event-log /slack-event-log
ENTRYPOINT ["/slack-event-log"]
