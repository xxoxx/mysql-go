# mysql-go
mysql-go

---
### 为什么有这个轮子
- mysql官方的命令行客户端没有静态编译的版本(网上也没找到) 
- 一般紧急临时用，mysql官方的命令行客户端体积很大，几百M
- 日常管理还是建议使用navicate之类的客户端

### Feature
- 类似linux下的grep,这样在管理过程中就少写一些 where,like等
```
mysql> show variables; | grep log_bin
Variable_name   Value
log_bin ON
log_bin_basename        /var/lib/mysql/binlog
log_bin_index   /var/lib/mysql/binlog.index
log_bin_trust_function_creators OFF
log_bin_use_v1_row_events       OFF
sql_log_bin     ON
6 rows in set (0.01 sec)
```
  mysql> show variables; | grep binlog
- 类似linux下的grep -v 排除过滤 
```
mysql> show processlist; | grepv sleep
Id      User    Host    db      Command Time    State   Info
5       event_scheduler localhost       NULL    Daemon  1902954 Waiting on empty queue  NULL
115     root    192.168.3.101:63598     NULL    Query   0       init    show processlist
2 rows in set (0.00 sec)
```

### 使用例子 基于跟官方的用法一至
```
mysql -uroot -p123456 -h127.0.0.1 -P3306
mysql -u root -p 123456 -h 127.0.0.1 -P 3306
mysql -uroot -p123456 -h127.0.0.1 -P3306 -Ddbname
mysql -uroot -p123456 -h127.0.0.1 -P3306 -f xxx.sql
mysql -uroot -p123456 -h127.0.0.1 -P3306 -Ddbname < xxx.sql
mysql -uroot -p123456 -h127.0.0.1 -P3306 -Ddbname -e 'select * from users limit 10;'
```

### 安装
```
wget https://gitee.com/tinatmp/mysql/releases/download/mysql/mysql_linux -O /usr/local/bin/mysql  chmod +x /usr/local/bin/mysql
```

### 其它
- 兼容 mysql5.7 mysql8 tidb
- 测试不全，可能存在bug