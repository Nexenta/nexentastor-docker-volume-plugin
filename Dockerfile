FROM alpine

RUN apk update

RUN mkdir -p /run/docker/plugins /var/lib/nvd/state

COPY bin/nvd /bin/nvd

CMD ["bin/sh"]
