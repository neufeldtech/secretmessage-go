FROM public.ecr.aws/docker/library/golang:1.20.5-alpine3.18 AS build

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o main .

FROM public.ecr.aws/docker/library/alpine:3.18

WORKDIR /app

COPY --from=build /app/main .

CMD ["./main"]