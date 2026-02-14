# vtuber-server + web 整体重构评审与改造方案（无历史兼容约束）

更新时间：2026-02-11  
范围：`vtuber-server`（Go）与 `vtuber-server/web/vtuber`（React/Electron/Web）

## 1. 评审输入与基线

本方案基于以下协议与实现对照：

- 协议基线：`mio/docs/xiaozhi-interaction-protocol.md`、`mio/docs/xiaozhi-websocket-protocol.md`
- 后端参考实现：`mio/backend-server/internal/server/transport/websocket/websocket_server.go`、`mio/backend-server/internal/server/chat/session/core.go`、`mio/backend-server/internal/server/chat/managers/session_state_machine.go`、`mio/backend-server/internal/server/chat/transport/server_transport.go`
- 当前待改造实现：`vtuber-server/internal/ws/handler.go`、`vtuber-server/pkg/xiaozhi/client.go`、`vtuber-server/web/vtuber/src/renderer/src/services/websocket-handler.tsx`、`vtuber-server/web/vtuber/src/renderer/src/services/websocket-service.tsx`

关键协议约束（来自 `mio/docs/xiaozhi-websocket-protocol.md` 与 `mio/docs/xiaozhi-interaction-protocol.md`）：

- 握手必须围绕 `hello/session_id/audio_params/Protocol-Version`。
- 控制面核心消息是 `listen(start|stop|detect, mode)`、`abort`、`mcp`。
- 音频面必须显式区分二进制版本（v1/v2/v3）与协商格式（opus/pcm_s16le/wav）。
- 实时模式需要独立于 auto/manual 的恢复语义（重点是连续监听与 TTS 期间行为）。

## 2. 当前实现核心问题（结论）

### 2.1 服务端（vtuber-server）

1. 会话编排、协议适配、业务能力耦合在单文件单类型，扩展成本高。  
证据：`vtuber-server/internal/ws/handler.go:169` 到文件末尾，`Handle + session` 同时承担接入、状态、音频、MCP、历史、分组等责任（文件总长 1633 行）。

2. 缺少独立状态机，行为由布尔标志拼接，跨事件一致性不稳定。  
证据：`vtuber-server/internal/ws/handler.go:986`（`ensureConversation/endConversation`）、`vtuber-server/internal/ws/handler.go:1012`（`handleTTS`）、`vtuber-server/internal/ws/handler.go:444`（`ensureListening`）。

3. 协议层与业务层边界不清晰，前后端自定义消息类型侵入核心链路。  
证据：`vtuber-server/internal/ws/handler.go:349` 的 `handleIncoming` 同时处理 `mic-audio-*`、`fetch-history-*`、`group-*`、`mcp-*`。

4. 预留架构壳未落地，导致“设计意图”和“实际代码”分离。  
证据：`vtuber-server/internal/conversation/orchestrator.go:1`、`vtuber-server/internal/conversation/state.go:1`、`vtuber-server/internal/media/audio.go:1`、`vtuber-server/internal/observability/metrics.go:1` 基本为空壳。

5. 与参考实现相比，缺少“传输层 / 状态机 / 会话处理”三层解耦。  
对照：`mio/backend-server/internal/server/transport/websocket/websocket_server.go:120`（接入层）、`mio/backend-server/internal/server/chat/session/core.go:1244`（会话语义）、`mio/backend-server/internal/server/chat/managers/session_state_machine.go:54`（状态机）。

### 2.2 前端（web/vtuber）

1. WebSocket 链路存在双入口与职责重复。  
证据：`vtuber-server/web/vtuber/src/renderer/src/context/websocket-context.tsx:98` 与 `vtuber-server/web/vtuber/src/renderer/src/services/websocket-handler.tsx:58` 同时处理 wsUrl/baseUrl 与 connect/reconnect 语义。

2. `WebSocketHandler` 过大，承载协议分发、UI状态、音频控制、媒体捕获、群组、历史等多领域逻辑。  
证据：`vtuber-server/web/vtuber/src/renderer/src/services/websocket-handler.tsx:58`（487 行，超大 switch）。

3. 应用状态分散在多层 Context Provider，状态一致性和调试成本高。  
证据：`vtuber-server/web/vtuber/src/renderer/src/App.tsx:198` 起的深层 Provider 嵌套。

4. 连接层缺少类型化协议约束，`MessageEvent` 中 `any` 字段大量存在。  
证据：`vtuber-server/web/vtuber/src/renderer/src/services/websocket-service.tsx:53` 起。

5. 存在渲染函数内直接写入全局 DOM 样式等副作用，维护风险高。  
证据：`vtuber-server/web/vtuber/src/renderer/src/App.tsx:62` 到 `:69`。

## 3. 目标架构（不考虑历史兼容）

## 3.1 设计原则

- 单一真相源：协议、消息类型、状态机都要单点定义。
- 明确边界：Transport（收发）/ Session（会话状态）/ Domain（功能）/ ViewModel（前端）分层。
- 协议优先：以前后端共享 schema 生成类型，禁止“字符串 + any”驱动主流程。
- 事件驱动：所有状态迁移由事件触发并可追踪。

### 3.2 服务端目标分层

建议目录重组（示意）：

```text
vtuber-server/internal/
  transport/
    clientws/             # 浏览器 <-> vtuber ws 接入
    xiaozhiws/            # vtuber <-> backend xiaozhi ws client
    codec/                # v1/v2/v3 binary codec + json envelope
  session/
    fsm/                  # 状态机（idle/listening/asr/llm/tts/...）
    runtime/              # 每会话运行时（队列、计时器、并发控制）
    dispatcher/           # 消息分发到 usecase
  usecase/
    dialog/               # text/audio/listen/interrupt 主链路
    mcp/                  # mcp initialize/list/call
    history/              # chat history
    group/                # group 管理
  adapter/
    vision/
    storage/
  observability/
    metrics/
    tracing/
```

关键点：

- `internal/ws/handler.go` 拆分为“接入层 + dispatcher + usecase”。
- `pkg/xiaozhi/client.go` 保留为 transport client，但只负责协议与连接，不承担会话业务。
- 使用显式 FSM 管理状态，参考 `backend-server` 的 `SessionStateMachine` 机制（状态迁移 + 进入/退出副作用）。

### 3.3 前端目标分层

建议目录重组（示意）：

```text
web/vtuber/src/renderer/src/
  protocol/
    schema.ts            # 共享协议定义（由 schema 生成）
    encode.ts
    decode.ts
  transport/
    ws-client.ts         # 连接、重连、心跳、发送队列
  domain/
    conversation/
    audio/
    history/
    group/
    mcp/
  store/
    app-store.ts         # 单一状态容器（zustand/redux）
  features/
    ...                  # UI 组件仅消费 store selectors
```

关键点：

- 取消 `websocket-context + websocket-handler` 双入口，保留一个连接管理器。
- 将“消息分发 switch”改成 typed reducer + handler registry。
- Context 仅保留跨切面能力（主题、国际化）；业务状态统一进 store。

## 4. 协议与状态机统一方案

### 4.1 协议统一

- 定义单一协议仓（建议 `protocol/messages.schema.json` + codegen）。
- 统一三类消息：
  - `ClientCommand`（前端 -> vtuber-server）
  - `ServerEvent`（vtuber-server -> 前端）
  - `XiaoZhiFrame/Message`（vtuber-server <-> backend-server）
- 删除非规范控制字符串（如 `control: conversation-chain-start/end`），替换为结构化事件：
  - `conversation.started`
  - `conversation.ended`
  - `audio.playback.completed`

### 4.2 状态机统一

服务端和前端都采用同一状态图（名称可差异但语义一致）：

- `idle`
- `listening`
- `processing_asr`
- `processing_llm`
- `sending_tts`
- `waiting_tts_complete`
- `interrupted`

迁移规则：

- 仅状态机可写入“当前会话状态”。
- 所有恢复动作（auto/realtime）走显式事件，不走隐式布尔判断。
- 统一 `listen_mode` 策略：`auto/manual/realtime` 在状态机中以策略对象处理。

## 5. 分阶段实施计划

### 阶段 A：协议与类型系统先行（1 周）

- 建立 schema 与 codegen，替换 Go/TS 内 `map[string]any` 和 `any` 主路径。
- 抽离 `xiaozhi codec`（v1/v2/v3）为独立包并补齐单测。

交付物：

- `internal/transport/codec/*`
- `web/.../protocol/*`
- 协议一致性测试（Go + TS）

### 阶段 B：服务端会话内核重写（1-2 周）

- 引入 `session/fsm` 与 `session/runtime`。
- `handler.go` 收敛为接入与路由，不再承载业务。
- 历史、分组、MCP 从会话主链路中解耦为 usecase。

交付物：

- 新会话内核 + 迁移后的 usecase
- 端到端用例：text、mic、interrupt、mcp capture、history、group

### 阶段 C：前端状态与连接层重构（1-2 周）

- 引入单一 store。
- 将 `websocket-handler.tsx` switch 拆成 domain handlers。
- `App.tsx` Provider 大幅瘦身。

交付物：

- 新 transport + store + handlers
- UI 不变更前提下的行为一致性测试

### 阶段 D：可观测性与压测（0.5-1 周）

- 增加会话级 tracing id、状态迁移日志、音频吞吐指标。
- 压测项：连续 1h 会话、断线重连、realtime 连续对话。

交付物：

- 仪表盘与告警阈值
- 回归报告模板

## 6. 关键技术决策

1. **不做历史兼容**：新协议版本直接升级（建议内部版本号 `v2`），旧前端不再接入。  
2. **前后端共享 schema**：协议字段变更必须通过 codegen；禁止手写分散定义。  
3. **状态机中心化**：禁止直接在业务逻辑中修改 `listening/ttsActive/inConversation` 这类布尔态。  
4. **Transport 无业务**：连接/编解码/重试与会话语义解耦。  
5. **MCP 工具隔离**：`capture -> vision` 作为独立 pipeline，不阻塞主语音状态机。  

## 7. 风险与应对

- 风险：一次性重构范围大，回归面广。  
  应对：按阶段替换，先协议和类型，再内核，再前端。

- 风险：音频链路时序问题（realtime/manual）在重构中回归。  
  应对：建立固定回放测试集，按模式回归。

- 风险：前端多上下文迁移到单 store 时行为不一致。  
  应对：先保留原 UI 组件，替换数据来源，最后清理旧 Context。

## 8. 验收标准

- 代码结构：`handler.go` 拆分后单文件不超过 400 行，且无跨领域 switch。
- 协议一致性：Go/TS 类型由同一 schema 生成；CI 校验 schema 未同步即失败。
- 稳定性：
  - 30 分钟持续 realtime 对话无会话卡死。
  - 断线重连后 3 秒内恢复可交互。
  - TTS stop/start 无重复抖动（可观测指标验证）。
- 可测试性：
  - 服务端至少有状态机迁移单测 + 编解码单测 + 会话集成测试。
  - 前端至少有消息分发 reducer 测试 + ws transport 测试。

## 9. 与 backend-server 的可复用对齐点

可直接借鉴的设计思想（不是直接拷贝代码）：

- 入口路由与连接认证分离（参考 `websocket_server.go` / `routes.go`）。
- `ServerTransport` 作为协议发送适配层（参考 `server_transport.go`）。
- 独立状态机驱动 TTS 开停与恢复（参考 `session_state_machine.go`）。
- `ChatSession` 作为会话编排核心，循环与 manager 分层（参考 `core.go`）。

## 10. 建议下一步

1. 先落一个 `protocol schema` PR（无业务改动）。
2. 再落 `session/fsm` 与 `transport/codec` PR（后端内核）。
3. 最后落前端 `single-store + handler registry` PR。

