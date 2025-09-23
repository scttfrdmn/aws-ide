FROM alpine:3.18

RUN apk --no-cache add ca-certificates git openssh-client && \
    addgroup -g 1000 aws-jupyter && \
    adduser -D -s /bin/sh -u 1000 -G aws-jupyter aws-jupyter

WORKDIR /home/aws-jupyter

COPY aws-jupyter /usr/local/bin/
COPY environments/ /usr/local/share/aws-jupyter/environments/

RUN chmod +x /usr/local/bin/aws-jupyter

USER aws-jupyter

ENTRYPOINT ["aws-jupyter"]