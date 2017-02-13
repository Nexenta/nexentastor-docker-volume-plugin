FROM alpine

RUN apk update

RUN mkdir -p /run/docker/plugins /mnt/state /mnt/volumes

COPY nvd nvd

CMD ["bin/sh"]
