FROM golang:1.11 AS build_base

WORKDIR /src
COPY go.mod .

RUN go mod download

FROM build_base AS builder

WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build -a -installsuffix cgo -ldflags="-w -s" -o /go/bin/touchngo-ynab-sync

FROM scratch

ENV YNAB_ACCESS_TOKEN ""
ENV YNAB_BUDGET_ID ""
ENV YNAB_ACCOUNT_ID ""
ENV YNAB_TOUCHNGO_CATEGORY_ID ""
ENV TOUCHNGO_URL ""
ENV TOUCHNGO_USERNAME ""
ENV TOUCHNGO_PASSWORD ""
ENV TOUCHNGO_CARD_SERIAL_NUMBER ""
ENV INSECURE "false"

COPY --from=builder /go/bin/touchngo-ynab-sync /bin/touchngo-ynab-sync

ENTRYPOINT ["/bin/touchngo-ynab-sync"]
