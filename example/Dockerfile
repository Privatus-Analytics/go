# Conformance example app (net/http + html/template). Build with the package
# folder as the context: docker build -f example/Dockerfile .
FROM golang:1.23 AS build
WORKDIR /src
# The package from this repo's own source (see the replace line in go.mod).
COPY . .
WORKDIR /src/example
RUN CGO_ENABLED=0 go build -trimpath -o /out/example .

FROM scratch
COPY --from=build /out/example /example
EXPOSE 8080
USER 65534
ENTRYPOINT ["/example"]
