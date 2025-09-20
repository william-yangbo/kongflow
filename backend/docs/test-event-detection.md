# 测试事件自动识别功能

## 📋 功能概述

测试事件自动识别功能提供了智能的事件分类机制，可以自动区分测试事件和生产事件，提升开发环境体验。这个功能完全对齐 trigger.dev 的实现模式。

## 🎯 解决的问题

- **测试数据污染：** 开发时的测试事件混入真实数据
- **调试困难：** 无法快速过滤掉测试事件
- **统计不准确：** 分析和监控数据包含测试噪音
- **UI 体验差：** 开发者在界面上看到大量测试数据

## 🔧 识别规则

### 优先级顺序

1. **显式测试标志 (最高优先级)**
2. **环境类型判断**
3. **API Key 前缀判断**
4. **事件名称模式匹配**

### 详细规则

#### 1. 显式测试标志

```go
// 通过 SendEventOptions.Test 字段显式指定
opts := &SendEventOptions{
    Test: &true,  // 强制标记为测试事件
}
```

#### 2. 环境类型判断

```go
// DEVELOPMENT 环境的事件默认为测试事件
env.Environment.Type == apiauth.EnvironmentTypeDevelopment
```

#### 3. API Key 前缀判断

```go
// 开发环境 API Key 前缀的事件标记为测试
strings.HasPrefix(apiKey, "tr_dev_") || strings.HasPrefix(apiKey, "pk_dev_")
```

#### 4. 事件名称模式

```go
// 以 "test." 开头的事件名称
strings.HasPrefix(event.Name, "test.")
```

## 💻 使用示例

### 1. 显式指定测试事件

```go
event := &SendEventRequest{
    ID:   "evt_12345",
    Name: "user.created",
}

opts := &SendEventOptions{
    Test: &true, // 显式标记为测试事件
}

response, err := eventsService.IngestSendEvent(ctx, env, event, opts)
// response.IsTest == true
```

### 2. 开发环境自动识别

```go
// 在 DEVELOPMENT 环境中
event := &SendEventRequest{
    ID:   "evt_12345",
    Name: "user.created",
}

response, err := eventsService.IngestSendEvent(ctx, devEnv, event, nil)
// response.IsTest == true (自动识别为测试事件)
```

### 3. 生产环境中的测试事件

```go
// 在 PRODUCTION 环境中
event := &SendEventRequest{
    ID:   "evt_12345",
    Name: "test.user.created", // test. 前缀
}

response, err := eventsService.IngestSendEvent(ctx, prodEnv, event, nil)
// response.IsTest == true (根据事件名称识别)
```

### 4. 强制非测试事件

```go
// 即使在开发环境，也可以强制标记为非测试事件
event := &SendEventRequest{
    ID:   "evt_12345",
    Name: "user.created",
}

opts := &SendEventOptions{
    Test: &false, // 强制标记为非测试事件
}

response, err := eventsService.IngestSendEvent(ctx, devEnv, event, opts)
// response.IsTest == false (显式标志优先级最高)
```

## 🧪 测试用例

功能包含全面的测试覆盖：

- ✅ 优先级测试：显式标志 > 环境类型 > API Key > 事件名称
- ✅ 环境类型测试：DEVELOPMENT/STAGING/PRODUCTION/PREVIEW
- ✅ API Key 前缀测试：tr*dev*/pk*dev*/tr*prod*/pk*prod*
- ✅ 事件名称测试：test.开头的事件
- ✅ 复合条件测试：多个规则组合
- ✅ 边界情况测试：空值、nil、特殊格式

## 📊 影响

### 数据库变更

- `event_records.is_test` 字段现在会根据规则自动设置
- 兼容现有数据结构，不需要迁移

### API 变更

- `SendEventOptions` 添加了 `Test *bool` 字段
- `EventRecordResponse` 的 `IsTest` 字段现在会正确反映事件类型

### 开发体验提升

- ✅ 开发环境事件自动标记为测试事件
- ✅ UI 可以基于 `isTest` 字段过滤显示
- ✅ 分析和统计可以排除测试数据
- ✅ 更清晰的开发调试体验

## 🔄 与 trigger.dev 对齐

此实现完全对齐 trigger.dev 的测试事件识别机制：

- 相同的优先级逻辑
- 相同的环境类型判断
- 相同的 API Key 前缀规则
- 相同的事件名称模式匹配
- 相同的 JSON 响应格式

## 📝 实现文件

- `internal/services/events/service.go` - 核心识别逻辑
- `internal/services/events/models.go` - 数据结构定义
- `internal/services/events/test_event_detection_test.go` - 测试用例

## 🚀 后续增强

可考虑的未来功能：

1. **UI 过滤器：** 基于 `isTest` 字段的前端过滤组件
2. **分析排除：** 统计报表自动排除测试事件
3. **测试模式：** 全局测试模式开关
4. **事件标签：** 更灵活的事件分类标签系统
