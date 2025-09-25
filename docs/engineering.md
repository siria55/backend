## 技术栈

### 接口

- API 网关 Gin：REST + WebSocket 推送

### 核心

- Agent Service
    - MockAgent：规则/脚本策略（课堂稳定演示用）
    - LLM，会真是调用大模型的地方
- Game Service

### 存储

- Postgres（任务/复盘/排行），Redis（会话态与队列）
