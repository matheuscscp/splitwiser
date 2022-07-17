FROM golang:1.18 AS build

WORKDIR /workspace
ADD go.mod .
ADD go.sum .
RUN go mod download
ADD *.go ./
RUN go build

FROM scratch

WORKDIR /app
COPY --from=build /lib64/ld-linux-x86-64.so.2 /lib64/ld-linux-x86-64.so.2
COPY --from=build /lib/x86_64-linux-gnu/libpthread.so.0 /lib/x86_64-linux-gnu/libpthread.so.0
COPY --from=build /lib/x86_64-linux-gnu/libc.so.6 /lib/x86_64-linux-gnu/libc.so.6
COPY --from=build /lib/x86_64-linux-gnu/libtinfo.so.6 /lib/x86_64-linux-gnu/libtinfo.so.6
COPY --from=build /lib/x86_64-linux-gnu/libdl.so.2 /lib/x86_64-linux-gnu/libdl.so.2
COPY --from=build /workspace/splitwiser /app/splitwiser
COPY --from=build /bin/bash /bin/bash
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ADD load-envs.sh .
ADD start-bot.sh .
ENTRYPOINT ["./start-bot.sh"]
