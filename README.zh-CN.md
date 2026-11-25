<div align="center">
  <h1>MCA-Cascade</h1>
  <p><strong>为 Cascade 宿主提供清晰的分配决策。</strong></p>
  <p><a href="./README.md">English</a></p>
</div>

MCA-Cascade 是一个仅依赖 Go 标准库的决策与客户端库，用于把 Cascade 宿主接入 MatchCore Arena 分配，为宿主集成提供精简而明确的认证查询与路由决策边界。

## 为什么选择 MCA-Cascade

- **依赖极少。** 整个库仅使用 Go 标准库。
- **认证内建。** 分配请求使用 OPS Bearer 令牌。
- **输入有界。** 响应解析对不受信任的数据设置明确边界。
- **决策纯粹。** 根据分配数据与目标映射生成结果，不产生宿主副作用。
- **失败如实。** 服务不可用时返回明确错误，不会伪装成“无分配”。
- **职责分离。** 本库不加载配置，也不进行文件 I/O。

## 进一步了解

- [Armorial Wiki](https://wiki.mcnb.dev/) — 生态文档
- [Armorial Contract](https://github.com/CoreMatch/HA-Contract) — 接口契约
- [Cascade](https://github.com/CoreMatch/Cascade) — 宿主代理项目
- [MatchCoreArena-VelocityGate](https://github.com/CoreMatch/MatchCoreArena-VelocityGate) — Velocity 侧路由插件

## 适用范围

宿主接线由库外负责；MCA-Cascade 不是完整的 Cascade 插件，也不作为生产部署方案呈现。

## 许可证

依据 [AGPL-3.0](LICENSE) 授权。
