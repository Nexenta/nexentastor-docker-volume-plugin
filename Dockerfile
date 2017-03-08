FROM alpine

RUN apk update

RUN mkdir -p /run/docker/plugins /mnt

COPY bin/nvd /bin/nvd

CMD ["bin/sh"]
