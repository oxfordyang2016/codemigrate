# Email 请求
## 获取
```
    curl -H "x-cydex-userlevel:9" -H "content-type: application/json" http://127.0.0.1:9000/api/v1/notification/email
```

## 设置
```
    curl -H "x-cydex-userlevel:9" -H "content-type: application/json" -X POST -d '{"enable":true, "contact_name":"bbb", "smtp_server":"1"}' http://127.0.0.1:9000/api/v1/notification/email
```

## 获取smtp服务器列表
```
    curl -H "x-cydex-userlevel:9" -H "content-type: application/json" http://127.0.0.1:9000/api/v1/notification/email/smtp_servers
```

## 获取单个smtp服务器
```
    curl -H "x-cydex-userlevel:9" -H "content-type: application/json" http://127.0.0.1:9000/api/v1/notification/email/smtp_servers/default
```

## 设置单个smtp服务器
```
    curl -H "x-cydex-userlevel:9" -H "content-type: application/json" -X POST -d '{"host":"test_h","port":1234,"account":"account@test.com","password":"123456"}' http://127.0.0.1:9000/api/v1/notification/email/smtp_servers/1
```
