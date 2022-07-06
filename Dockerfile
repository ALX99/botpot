FROM alpine:latest

COPY ./build/botpot /botpot


CMD [ "/botpot" ]