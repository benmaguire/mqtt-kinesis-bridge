FROM golang:1.17-alpine as builder
RUN apk add git

WORKDIR /app

COPY . .

RUN go get . && \
    CGO_ENABLED=0 go build -a -installsuffix cgo \
	-ldflags "-s -w" \
	-o mqtt-kinesis-bridge .


# Use distroless as minimal base image to package the binary
FROM gcr.io/distroless/static:nonroot

COPY --from=builder /app/mqtt-kinesis-bridge /
USER nonroot:nonroot

CMD ["/mqtt-kinesis-bridge"]
