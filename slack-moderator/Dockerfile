FROM alpine
RUN apk update && apk add tzdata ca-certificates
ADD slack-moderator /slack-moderator
ENTRYPOINT ["/slack-moderator"]
