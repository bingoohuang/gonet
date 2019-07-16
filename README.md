# gonet

net relative like port, http, rest.

1. FreePort
1. Get/Post/Put/Patch/Delete
1. TLS relatives
1. ListLocalIps


## Make certs

参见[cert_test.sh](./cert_test.sh)

```bash
openssl genrsa -out root.key 2048
openssl req -new -nodes -x509 -days 3650 -key root.key -out root.pem -subj "/C=CN/ST=BEIJING/L=Earth/O=BJCA/OU=IT/CN=root"

openssl genrsa -out server.key 2048
openssl req -new -key server.key -out server.csr -subj "/C=CN/ST=BEIJING/L=Earth/O=BJCA/OU=IT/CN=server"
openssl x509 -req -in server.csr -CA root.pem -CAkey root.key -CAcreateserial -out server.pem -days 3650

openssl genrsa -out client.key 2048
openssl req -new -key client.key -subj "/C=CN/ST=BEIJING/L=Earth/O=BJCA/OU=IT/CN=client" -out client.csr
echo "[ssl_client]"> openssl.cnf; echo "extendedKeyUsage = clientAuth" >> openssl.cnf;
openssl x509 -req -in client.csr -CA root.pem -CAkey root.key -CAcreateserial -extfile ./openssl.cnf -out client.pem -days 3650
```

## Thanks

1. [urllib](https://github.com/GiterLab/urllib)
1. [go-resty](https://github.com/go-resty/resty/tree/v2)
1. [sling](https://github.com/dghubble/sling)
1. [This demonstrates how to make client side certificates with go](https://gist.github.com/ncw/9253562)
1. [带入gRPC：基于 CA 的 TLS 证书认证](https://studygolang.com/articles/15331)
1. [Better self-signed certificates](https://github.com/Shyp/generate-tls-cert)
1. [Create a PKI in GoLang](https://fale.io/blog/2017/06/05/create-a-pki-in-golang/)
1. [root CA and VerifyClientCert](https://play.golang.org/p/NyImQd5Xym)
1. [Golang的TLS证书](https://blog.csdn.net/fyxichen/article/details/51250620)
1. [密钥、证书生成和管理总结](https://www.cnblogs.com/pixy/p/4722381.html)
1. [TLS with Go](https://ericchiang.github.io/post/go-tls/)
1. [Golang - Go与HTTPS](http://www.golangtab.com/2018/02/05/Golang-Go与HTTPS/)
