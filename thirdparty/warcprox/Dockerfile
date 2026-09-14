FROM python:3.12-slim

ARG WARCPROX_VERSION=2.13.1

RUN pip install --no-cache-dir "warcprox==${WARCPROX_VERSION}"

EXPOSE 8000

ENTRYPOINT ["warcprox"]
