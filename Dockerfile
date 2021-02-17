FROM alpine:3.13
ARG FULLNAME="Agile Stacks"
LABEL maintainer="${FULLNAME} <support@agilestacks.com>"
COPY bin/linux/hub /usr/local/bin/hub
