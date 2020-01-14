# ipv6 support

## [ipv6-system-support](https://dev.mysql.com/doc/refman/5.7/en/ipv6-system-support.html)

For example, this command connects over IPv6 to the MySQL server on the local host: 

```bash
shell> mysql -h ::1
mysql> CREATE USER 'bill'@'::1' IDENTIFIED BY 'secret'; 
mysql> GRANT SELECT ON mydb.* TO 'bill'@'::1';
```

[5.1.11.1 Verifying System Support for IPv6](https://dev.mysql.com/doc/refman/5.7/en/ipv6-system-support.html)

```bash
shell> ping6 ::1 
16 bytes from ::1, icmp_seq=0 hlim=64 time=0.171 ms 
16 bytes from ::1, icmp_seq=1 hlim=64 time=0.077 ms 
...
```

## [Adding IPv6 to a keepalived and haproxy cluster](https://raymii.org/s/articles/Adding_IPv6_to_a_keepalived_and_haproxy_cluster.html)

`netstat -tlpn | grep haproxy`

## [IPv6格式](https://zh.wikipedia.org/wiki/IPv6)

IPv6二进位制下为128位长度，以16位为一组，每组以冒号“:”隔开，可以分为8组，每组以4位十六进制方式表示。

例如：2001:0db8:86a3:08d3:1319:8a2e:0370:7344 是一个合法的IPv6地址。

类似于IPv4的点分十进制，同样也存在点分十六进制的写法，将8组4位十六进制地址的冒号去除后，每位以点号“.”分组，

例如：2001:0db8:85a3:08d3:1319:8a2e:0370:7344则记为2.0.0.1.0.d.b.8.8.5.a.3.0.8.d.3.1.3.1.9.8.a.2.e.0.3.7.0.7.3.4.4，
其倒序写法用于ip6.arpa子域名记录IPv6地址与域名的映射。
同时IPv6在某些条件下可以省略：

1. 每项数字前导的0可以省略，省略后前导数字仍是0则继续，例如下组IPv6是等价的。
    - 2001:0DB8:02de:0000:0000:0000:0000:0e13
    - 2001:DB8:2de:0000:0000:0000:0000:e13
    - 2001:DB8:2de:000:000:000:000:e13
    - 2001:DB8:2de:00:00:00:00:e13
    - 2001:DB8:2de:0:0:0:0:e13
1. 可以用双冒号“::”表示一组0或多组连续的0，但只能出现一次：
    - 如果四组数字都是零，可以被省略。遵照以上省略规则，下面这两组IPv6都是相等的。
    - 2001:DB8:2de:0:0:0:0:e13
        - 2001:DB8:2de::e13
    - 2001:0DB8:0000:0000:0000:0000:1428:57ab
        - 2001:0DB8:0000:0000:0000::1428:57ab
        - 2001:0DB8:0:0:0:0:1428:57ab
        - 2001:0DB8:0::0:1428:57ab
        - 2001:0DB8::1428:57ab
    - 2001::25de::cade 是非法的，因为双冒号出现了两次。它有可能是下种情形之一，造成无法推断。
        - 2001:0000:0000:0000:0000:25de:0000:cade
        - 2001:0000:0000:0000:25de:0000:0000:cade
        - 2001:0000:0000:25de:0000:0000:0000:cade
        - 2001:0000:25de:0000:0000:0000:0000:cade
    - 如果这个地址实际上是IPv4的地址，后32位可以用10进制数表示；因此::ffff:192.168.89.9 相等于::ffff:c0a8:5909。

另外，::ffff:1.2.3.4 格式叫做IPv4映射地址。

IPv4位址可以很容易的转化为IPv6格式。举例来说，如果IPv4的一个地址为135.75.43.52（十六进制为0x874B2B34），它可以被转化为0000:0000:0000:0000:0000:FFFF:874B:2B34 或者::FFFF:874B:2B34。同时，还可以使用混合符号（IPv4-compatible address），则地址可以为::ffff:135.75.43.52。

链路本地地址

- ::1/128－是一种单播绕回地址。如果一个应用程序将数据包送到此地址，IPv6堆栈会转送这些数据包绕回到同样的虚拟接口（相当于IPv4中的127.0.0.1/8）。

## [How do ports work with IPv6?](https://stackoverflow.com/questions/186829/how-do-ports-work-with-ipv6)

For example : `http://[1fff:0:a88:85a3::ac1f]:8001/index.html`

[The notation in that case is to encode the IPv6 IP number in square brackets](https://serverfault.com/questions/205793/how-can-one-distinguish-the-host-and-the-port-in-an-ipv6-url
):

`http://[2001:db8:1f70::999:de8:7648:6e8]:100/`

That's RFC 3986, section 3.2.2: Host

A host identified by an Internet Protocol literal address, version 6 [RFC3513] or later, 
is distinguished by enclosing the IP literal within square brackets ("[" and "]"). 
This is the only place where square bracket characters are allowed in the URI syntax. 
In anticipation of future, as-yet-undefined IP literal address formats, 
an implementation may use an optional version flag to indicate such a format explicitly rather than 
rely on heuristic determination.



## [haproxy](https://raymii.org/s/articles/Adding_IPv6_to_a_keepalived_and_haproxy_cluster.html)

haproxy is suprisingly easy with IPv6. Just add it to your frontend section as a bind option:


```haproxy
frontend http-in
      mode http
      bind 1.2.3.4:80
      bind 2a02:123:45:67bb::1:80 transparent
      option httplog
      option forwardfor
      option http-server-close
      option httpclose
      reqadd X-Forwarded-Proto:\ http
      http-request add-header X-Real-IP %[src]
      default_backend appserver
```
You must add the transparant option. Otherwise, haproxy will not start if the VIP is not on the machine itself. (kind of like nonlocal.bind sysctl).

> haproxy is intelligent enough to understand the port number in the address. No need to screw around with brackets like `[2a02:123:45:67bb::1]:80` or special options.



## [haproxy Transparent Proxy](https://www.haproxy.com/blog/howto-transparent-proxying-and-binding-with-haproxy-and-aloha-load-balancer/)  

### Transparent Proxy

Here comes the transparent proxy mode: HAProxy can be configured to spoof the client IP address when establishing the TCP connection to the server. That way, the server thinks the connection comes from the client directly (of course, the server must answer back to HAProxy and not to the client, otherwise it can’t work: the client will get an acknowledge from the server IP while it has established the connection on HAProxy‘s IP).

HAProxy and the Linux Kernel
Unfortunately, HAProxy can’t do transparent binding or proxying alone. It must stand on a compiled and tuned Linux Kernel and operating system.
Below, I’ll explain how to do this in a standard Linux distribution.
Here is the check list to meet:
1. appropriate HAProxy compilation option
2. appropriate Linux Kernel compilation option
3. sysctl settings
4. iptables rules
5. ip route rules
6. HAProxy configuration

HAProxy compilation requirements
First of all, HAProxy must be compiled with the option TPROXY enabled.
It is enabled by default when you use the target LINUX26 or LINUX2628.


## MySQL driver

golang driver [TCP via IPv6](https://github.com/go-sql-driver/mysql)

`user:password@tcp([de:ad:be:ef::ca:fe]:80)/dbname?timeout=90s&collation=utf8mb4_unicode_ci`

java driver ipv4

```java
urlString = "jdbc:mysql://10.144.1.216:3306/dbName";
Class.forName(driver);
DriverManager.setLoginTimeout(getConnectionTimeOut());
dbConnection = DriverManager.getConnection(urlString,user,password);
```

[java driver ipv6](http://blog.ashwani.co.in/blog/2012-10-10/mysql-with-ipv6/)

```java
urlString = "jdbc:mysql://address=(protocol=tcp)(host=fe80::5ed6:baff:fe14:a23e)(port=3306)/db";
```

## MySQL Server

[bind_address](https://dev.mysql.com/doc/refman/5.7/en/server-system-variables.html#sysvar_bind_address)

Property |	Value
---|---
Command-Line Format	| --bind-address=addr
System Variable	| bind_address
Scope	|Global
Dynamic	|No
Type	|String
Default Value	|*

The MySQL server listens on a single network socket for TCP/IP connections. This socket is bound to a single address, but it is possible for an address to map onto multiple network interfaces. To specify an address, set bind_address=addr at server startup, where addr is an IPv4 or IPv6 address or a host name. If addr is a host name, the server resolves the name to an IP address and binds to that address. If a host name resolves to multiple IP addresses, the server uses the first IPv4 address if there are any, or the first IPv6 address otherwise.

The server treats different types of addresses as follows:

1. If the address is *, the server accepts TCP/IP connections on all server host IPv4 interfaces, and, if the server host supports IPv6, on all IPv6 interfaces. Use this address to permit both IPv4 and IPv6 connections on all server interfaces. This value is the default.
1. If the address is 0.0.0.0, the server accepts TCP/IP connections on all server host IPv4 interfaces.
1. If the address is ::, the server accepts TCP/IP connections on all server host IPv4 and IPv6 interfaces.