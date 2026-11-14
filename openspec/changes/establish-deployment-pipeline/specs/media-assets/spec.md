# Spec Delta

## ADDED Requirements

### Requirement: 幼儿媒体通过短期签名 URL 访问

媒体文件 SHALL 存储在私有桶中；系统 MUST NOT 暴露长期有效的公共 URL。读取接口 SHALL 返回有界有效期的签名 GET URL（默认 1 小时）；数据库 SHALL 保存 object key 而非完整 URL；客户端在 URL 过期后 MUST 重新获取。

#### Scenario: 读取返回短期签名 URL

- **WHEN** 客户端请求一条成长记录或相册中的媒体
- **THEN** 响应中的 URL 为带有效期的签名 URL
- 参考: TC-NEW-MAS-0012

#### Scenario: 过期 URL 被拒绝并可重新获取

- **WHEN** 客户端使用已过期的签名 URL 请求对象
- **THEN** 存储服务拒绝访问；客户端重新获取后得到新的可用签名 URL
- 参考: TC-NEW-MAS-0013

#### Scenario: 未签名公共 URL 不可访问

- **WHEN** 不使用签名直接请求对象 URL
- **THEN** 存储服务拒绝访问
- 参考: TC-NEW-MAS-0014
