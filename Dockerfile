FROM golang:1.8.3-alpine3.5
COPY ./ /go/src/WeChatTokenRefresh/
WORKDIR /go/src/WeChatTokenRefresh/
RUN go install .

FROM terrysolar/alpine:SSL_TIMEZONE_FIX

ENV WECHAT_APP_ID=""
ENV WECHAT_APP_SECRET=""
ENV REDIS_ADDR=""
ENV REDIS_PASSWORD=""
ENV REDIS_DB=0
ENV REDIS_KEY="wechat:access_token"
ENV REDIS_JS_KEY="wechat:jsapi_ticket"
ENV RETRY_TIMES=5
ENV TICK=7160
ENV MAIL_ALERT="false"
ENV MAIL_SMTP_SERVER=""
ENV MAIL_SMTP_PORT=465
ENV MAIL_ACCOUNT=""
ENV MAIL_PASSWORD=""
ENV MAIL_RECEIVE_ACCOUNT=""

COPY --from=0 /go/bin/WeChatTokenRefresh /

RUN chmod +x /WeChatTokenRefresh

ENTRYPOINT [ "/WeChatTokenRefresh" ]


