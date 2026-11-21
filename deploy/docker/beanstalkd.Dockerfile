# Build from a maintained Alpine package; publish an immutable digest before deployment.
FROM alpine:3.22
RUN apk add --no-cache beanstalkd && mkdir -p /data && chown 10001:10001 /data
USER 10001:10001
VOLUME /data
EXPOSE 11300
ENTRYPOINT ["/usr/bin/beanstalkd"]
CMD ["-l","0.0.0.0","-p","11300","-b","/data","-f","1000"]
