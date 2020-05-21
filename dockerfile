FROM gcr.io/distroless/static
MAINTAINER Daniel Randall <danny_randall@byu.edu>

ARG NAME

COPY ${NAME} /pc-config

ENTRYPOINT ["/pc-config"]
