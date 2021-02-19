FROM alpine:3.13
ARG FULLNAME="Agile Stacks"
LABEL maintainer="${FULLNAME} <support@agilestacks.com>"
RUN \
    apk update && apk upgrade && \
    apk add --no-cache \
        git \
        git-subtree && \
    rm -rf /var/cache/apk/*
COPY bin/linux/hub /usr/local/bin/hub
