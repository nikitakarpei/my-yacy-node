FROM golang:1.26-alpine AS build

RUN apk add --no-cache git

ARG SMOKESCREEN_REPO=https://github.com/stripe/smokescreen.git
ARG SMOKESCREEN_COMMIT=a3294a6cc4e4e33cd086ec0d5bca9f22db097023

WORKDIR /src
RUN git init \
    && git remote add origin "${SMOKESCREEN_REPO}" \
    && git fetch --depth 1 origin "${SMOKESCREEN_COMMIT}" \
    && git checkout FETCH_HEAD

RUN CGO_ENABLED=0 go build -o /smokescreen .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates

COPY --from=build /smokescreen /usr/local/bin/smokescreen

EXPOSE 4750
ENTRYPOINT ["smokescreen"]
