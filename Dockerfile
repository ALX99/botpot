FROM golang:1.19 as build

WORKDIR /wrk
COPY . .
RUN CGO_ENABLED=0 go build -o /build/botpot -trimpath -ldflags "-s -w" ./cmd/botpot/main.go

FROM gcr.io/distroless/static-debian11
COPY --from=build /build/botpot /

CMD ["/botpot"]
