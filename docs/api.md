# API 文档

## 健康检查
- **方法**：GET
- **路径**：`/healthz`
- **说明**：返回服务运行状态与基础环境信息，可用于探活。
- **示例响应**：

```json
{
  "status": "ok",
  "environment": "development",
  "dependencies": ["agent", "game"]
}
```

## Agent 示例配置
- **方法**：GET
- **路径**：`/v1/agents/mock`
- **说明**：返回火星前哨站示例 Agent（阿瑞斯-01）的静态配置，用于前端调试或接口联调。
- **示例响应**：

```json
{
  "name": "ARES-01",
  "role": "Autonomous Research & Exploration Sentinel",
  "capabilities": ["Scan", "Route", "Sync"],
  "personality": "冷静务实，偏数据驱动",
  "primary_sensor": "复合光谱雷达"
}
```

## 游戏状态快照
- **方法**：GET
- **路径**：`/v1/game/state`
- **说明**：返回火星前哨站当前时刻的示例游戏状态，包括周期、人口与下一事件，用于前端状态展示。
- **示例响应**：

```json
{
  "cycle": "Sol-128",
  "population": 42,
  "next_event": "沙尘暴预警",
  "event_time": "2024-01-01T12:34:56Z",
  "environment": "低压、低重力、穹顶保护开启"
}
```

> `event_time` 字段值示例为 UTC 时间，真实运行时会根据当前时间计算。
