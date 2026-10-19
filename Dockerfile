FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go .
RUN CGO_ENABLED=0 go build -o /sample .

FROM gcr.io/distroless/static-debian12
ARG RELEASE=dev
ENV RELEASE=${RELEASE}
COPY --from=build /sample /sample
EXPOSE 8080
CMD ["/sample", "serve"]
