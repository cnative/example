# NOTE: the artifacts that are copied below are expected to be prebuilt and be the same
# ones that are produced and unit-tested during the previous steps of the
# CI system, which ensures what is built is used in all stages of the testing.

FROM alpine:latest
RUN apk --update add ca-certificates jq
COPY bin/linux_amd64/cnative-example /usr/bin/cnative-example
COPY bin/linux_amd64/cnative-example-e2e-tests /usr/bin/cnative-example-e2e-tests
COPY tests/e2e/init.sh /usr/bin/init.sh

ENTRYPOINT ["/usr/bin/cnative-example-e2e-tests"]