FROM gcr.io/distroless/static-debian11

COPY  ./build/botpot /botpot

CMD ["/botpot"]
