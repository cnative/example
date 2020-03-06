FROM alpine:latest as certs
RUN apk --update add ca-certificates

FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY example-app /usr/bin/example-app
ENTRYPOINT ["/usr/bin/example-app"]
CMD ["-h"]