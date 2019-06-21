
https://colobu.com/2016/06/07/simple-golang-tls-examples/


1. 生成服务器端的私钥  `openssl genrsa -out server.key 2048`

    ```bash
    ➜  https git:(master) ✗ openssl genrsa -out server.key 2048
    Generating RSA private key, 2048 bit long modulus
    ..+++
    ...+++
    e is 65537 (0x10001)
    ```

1. 生成服务器端证书 `openssl req -new -x509 -key server.key -out server.pem -days 3650`

    ```bash
    ➜  https git:(master) ✗ openssl req -new -x509 -key server.key -out server.pem -days 3650
    You are about to be asked to enter information that will be incorporated
    into your certificate request.
    What you are about to enter is what is called a Distinguished Name or a DN.
    There are quite a few fields but you can leave some blank
    For some fields there will be a default value,
    If you enter '.', the field will be left blank.
    -----
    Country Name (2 letter code) []:CN
    State or Province Name (full name) []:Beijing
    Locality Name (eg, city) []:Beijing
    Organization Name (eg, company) []:www.bjca.cn
    Organizational Unit Name (eg, section) []:sa
    Common Name (eg, fully qualified host name) []:.
    Email Address []:.
    ```

1. 或者，同时生成私钥和证书 `openssl req -new -nodes -x509 -out server.pem -keyout server.key -days 3650 -subj "/C=CN/ST=Beijing/L=Beijing/O=www.bjca.cn/OU=IT/CN=./emailAddress=."


1. 生成客户端的私钥 `openssl genrsa -out client.key 2048`

    > 所谓的"客户端证书"就是用来证明客户端访问者的身份。
    > 比如在某些金融公司的内网，你的电脑上必须部署"客户端证书"，才能打开重要服务器的页面。

1. 生成客户端的证书 `openssl req -new -x509 -key client.key -out client.pem -days 3650`
1. 或者，同时生成私钥和证书 `openssl req -new -nodes -x509 -out client.pem -keyout client.key -days 3650 -subj "/C=DE/ST=Beijing/L=Beijing/O=www.bjca.cn/OU=IT/CN=./emailAddress=."
1. 查看证书  `openssl x509 -in server.pem -noout -text`
