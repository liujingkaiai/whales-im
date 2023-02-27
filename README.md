## 目录结构 
- container 容器层
- core 字节实现代码
- examples mock示例
- logger 日志
- iface 存放接口
- wire 协议层
- services 业务层

### Container 容器层
- 托管server 
- 维护服务依赖关系
- 处理消息上行下行



### 上行消息
- 链路   用户sdk->网关->逻辑服务

### 下行消息
- 链路   逻辑服务->网关->用户sdk
- 通过Push方法 把消息发送给网关