# go-pkg-tddcheck

`tddcheck` 用于检查 Go 项目的 handler/service/repository 分层约束。

## 使用方式

命令行运行：

```bash
go run ./cmd/tddcheck -root internal
```

在项目测试中引用：

```go
package tddcheck_test

import (
	"testing"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck"
)

func TestRules(t *testing.T) {
	tddcheck.ProjectRules{Root: "internal"}.Assert(t)
}
```

运行本地架构检查时建议使用：

```bash
go test -count=1 ./internal/testutil/tddcheck
```

## 文件命名

资源文件必须使用：

```text
{resource}.{type}.go
```

架构/共享文件必须使用：

```text
x_{scope}.{type}.go
```

示例：

```text
internal/handler/device.handler.go
internal/handler/device.dto.go
internal/handler/x_router.handler.go

internal/service/device.service.go
internal/service/device.commands.go
internal/service/x_shared.support.go

internal/repository/device.support.go
internal/repository/device.store.go
internal/repository/x_database.repository.go
```

拒绝示例：

```text
device_handler.go
shared.model.go
device_update.utils.go
device.models.go
device.writes.go
device.database.go
helper.utils.go
```

## 文件类型

各层允许的文件类型：

```text
handler:    dto, handler, mapper, utils
service:    commands, mapper, service, support
repository: repository, schema, store, support
```

各层允许的架构 scope：

```text
handler:    x_api, x_frontend, x_router, x_shared
service:    x_batch, x_id, x_shared
repository: x_database, x_schema, x_store, x_shared
```

允许的架构文件：

```text
handler:
  x_api.handler.go, x_api.utils.go
  x_frontend.handler.go, x_frontend.utils.go
  x_router.handler.go, x_router.utils.go
  x_shared.handler.go, x_shared.utils.go

service:
  x_batch.service.go
  x_id.support.go
  x_shared.mapper.go, x_shared.support.go

repository:
  x_database.repository.go
  x_schema.repository.go, x_schema.support.go
  x_store.repository.go
  x_shared.support.go
```

## 内容规则

```text
*.commands.go   只能声明类型；类型名必须以 Request、Response、Result 或 Item 结尾
*.dto.go        只能声明 DTO/DTOs 类型；不能声明函数
*.handler.go    只能声明 handler 结构体、Register* 函数和 handler 方法
*.mapper.go     只能声明包级 To* 函数
*.repository.go 只能用于 repository 架构文件
*.schema.go     只能声明 schema 生命周期函数
*.service.go    只能声明一个 Service 结构体、New*Service 和 service 方法
*.store.go      只能声明 Store 方法
*.support.go    可声明类型、const、Err* var、util*/validate*/normalize*/Wrap*/Is*/As* 函数
*.utils.go      只能声明包级 util* 函数
```

## 命名规则

```text
资源 scope 使用 snake_case
架构 scope 使用 x_ 前缀
禁止使用 common、default、helper、helpers、misc、util、utils 等弱 scope
资源 scope 不能包含 update、mapper、service、store、validation 等文件类型词
mapper 函数必须以 To 开头
utils 函数必须以 util 开头
```

## 分层依赖

默认禁止的 import：

```text
handler    -> repository
service    -> handler
repository -> handler
repository -> service
```
