FROM alpine:latest as certs
RUN apk --update add ca-certificates

FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY cnative-example /usr/bin/cnative-example
ENTRYPOINT ["/usr/bin/cnative-example"]
CMD ["-h"]