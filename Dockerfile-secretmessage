FROM golang:1.18

WORKDIR "/go/src/secretMessage"
COPY . .

RUN make build

EXPOSE 8080
CMD ["./secretmessage"]
