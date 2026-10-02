# Agent B01：模型 Provider 协议与配置

当前适配器实现非流式 **Chat Completions 的 function tools** 协议：请求携带 `model/messages/tools`，助手工具调用保存 `id/function.name/function.arguments`，工具结果通过 `tool_call_id` 回传；响应保留 `finish_reason` 和 token usage。协议形状依据 [OpenAI Chat API 参考](https://developers.openai.com/api/reference/cli/resources/chat) 与 [官方 function calling 指南](https://developers.openai.com/api/docs/guides/function-calling)。其他供应商只有明确支持同一协议时才能复用此适配器；当前未持有真实供应商密钥，兼容性和实际模型输出质量尚未联调。官方指南说明部分 OpenAI 模型的工具调用需改用 Responses API；此适配器不自动转换协议。

配置复用 `AI_ENABLED / AI_BASE_URL / AI_API_KEY / AI_MODEL / AI_TIMEOUT / AI_MAX_RETRIES`。Agent 关闭或只运行规则时不要求这些值。调用时缺少模型、地址、密钥或关闭 AI 会返回 `ErrLLMUnavailable`，不使服务启动失败。远程地址要求 HTTPS，本地回环地址允许 HTTP 供测试；禁止重定向传送授权头。授权头、供应商响应正文和原始请求正文均不进入错误信息。

单次 Provider 调用及其重试总计最多 25 秒，HTTP 响应体最多 1 MiB；429/5xx 和临时网络错误最多重试两次，4xx 参数/权限错误、畸形响应不重试。B03 工具循环还需给整轮对话施加 25 秒共享预算；目前 Provider 已接线，但现有聊天仍使用确定性降级，尚未调用模型。

无真实密钥的 `httptest.Server` 覆盖模型与地址、普通回复、单个和多个工具调用、工具结果回传、JSON 错误、429/5xx、400、体积限制、超时和取消。测试仅证明协议适配器行为，不代表任意第三方兼容服务或具体模型可用。
