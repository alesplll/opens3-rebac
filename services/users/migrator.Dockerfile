FROM alpine:3.21
RUN apk update && \
    apk upgrade && \
    apk add bash && \
    rm -rf /var/cache/apk/*

ADD https://github.com/pressly/goose/releases/download/v3.14.0/goose_linux_x86_64 /bin/goose
RUN chmod +x /bin/goose

WORKDIR /root

COPY migrations/*.sql migrations/
COPY .env .
COPY migration.sh .

RUN chmod +x migration.sh

ENTRYPOINT ["bash", "migration.sh"]
