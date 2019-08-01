# wechat-token

> Refresh your wechat access_token and jssdk ticket automatically

## QuickStart

```
docker run \
    -e WECHAT_APP_ID={your appid} \
    -e WECHAT_APP_SECRET={your app secret} \
    -e REDIS_ADDR={redis ip:port} \
    -e REDIS_PASSWORD={redis password} \
    -it --rm --name=wx-refresh terrysolar/wechat-token
    
【output】
2019/08/01 14:27:40 Success Access Token! 24_wa4HmkcIkjeGXJ_20C8X--zOWIoQ0d2C5X51BfmFMTN1UCohhY85Lu0tVbJbt8wBZ7sQWG5doFQDbpZbL_piYgk9Pb5fRnz8-VuErdnN5NLLiko9gFDC5KJwCFUKRBjABAUBL
2019/08/01 14:27:40 Success Js Ticket! sM4AOVdWfPE4DxkXGEs8VDt0ht6Y2mMwTlIw3Xb-g0mlaZvptxoONRFcfFg_yAhAATmJaQW12ZyH3x-cDEPPVA
```

## Environment variables

name | default value | description
---|--- | ---
WECHAT_APP_ID     |   | wechat app_id
WECHAT_APP_SECRET |   | wechat app_secret
REDIS_ADDR        |   | redis ip:port
REDIS_PASSWORD    |   | redis password
REDIS_DB          | 0 | redis db index
REDIS_KEY     | wechat:access_token | access_token redis key
REDIS_JS_KEY  | wechat:jsapi_ticket | jssdk ticket redis key
RETRY_TIMES       | 5 | retry times 
TICK              | 7160 | refresh time period
MAIL_ALERT        | false | need mail alert when refresh failed
MAIL_SMTP_SERVER  | | MAIL_SMTP_SERVER
MAIL_SMTP_PORT    | | MAIL_SMTP_PORT
MAIL_ACCOUNT      | | MAIL_ACCOUNT
MAIL_PASSWORD     | | MAIL_PASSWORD
MAIL_ALERT_RECEIVE_ACCOUNT | | MAIL_ALERT_RECEIVE_ACCOUNT