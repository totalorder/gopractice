FROM golang AS build
WORKDIR /got
COPY go.mod .
COPY hello.go .
RUN CGO_ENABLED=0 GOOS=linux go build -o hello .

FROM scratch
COPY --from=build /got/hello /hello
CMD ["/hello"]
