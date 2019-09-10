FROM golang

WORKDIR /einride
COPY . .
RUN go mod tidy
CMD ["go", "test", "."]