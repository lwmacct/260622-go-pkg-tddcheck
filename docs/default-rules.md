# tddcheck 默认规则

本文档描述 `tddcheck.DefaultConfig()` 提供的三包架构规则、扫描边界和配置方式。

## 设计边界

默认结构固定为三个 Go package：

```text
internal/handler
internal/service
internal/repository
```

每层可以包含多个业务 subject，但不会为 subject 创建子 package。业务边界主要由文件 identity、声明 ownership 和层间 import 共同表达：

```text
handler -> service -> repository
```

默认配置还会校验三个层目录是 analyzed root 的直接子目录，并要求 package 名分别为
`handler`、`service`、`repository`。自定义层只有显式配置 `LayerPackageNames` 时才启用
同样的单 package 契约。

默认规则负责：

```text
filelayout  文件 identity、文件角色、声明内容、subject ownership 和跨文件约束
layerdeps   当前 module 内的分层 import 约束
```

默认规则不验证数据库业务不变量、事务、锁、幂等参数或 API endpoint 契约。这些内容应由项目测试、数据库迁移或项目自定义规则负责。

## 执行模型

检查流程通过 `go/packages` 加载目标 module subtree。所有规则共享同一个 `token.FileSet`、package graph、AST、`types.Info` 和 imports 快照。

实际文件集合由 Go build constraints、`Config.BuildFlags` 和当前 GOOS/GOARCH 决定。package/type error 会保留在分析结果中，并使 `Analysis.Passed()` 失败；`Config.StrictPackages=true` 还会在加载阶段直接返回错误。

特殊文件和目录：

```text
*_test.go                                  默认不加载
*_test.go + IncludeTests=true              参与依赖规则和自定义规则；不参与默认 filelayout
{subject}.free.go / x.{namespace}.free.go  跳过声明和 ownership 规则，但产生 warning
隐藏目录                                   不参与扫描
vendor/node_modules/dist/build             默认不参与扫描
```

CLI 可通过 `--config tddcheck.json` 读取配置。JSON 使用 lowerCamel 字段名，并拒绝重复字段、未知字段和无效 UTF-8。

## 文件 Identity

默认层使用 `qualified_kind` 命名：

```text
{subject}.{kind}.go
x.{namespace}.{kind}.go
```

两种 identity 的职责不同：

```text
subject file       只承载一个业务 subject 的声明
architecture file  承载跨 subject 或基础设施声明
```

`subject` 和 `namespace` 是 lowercase snake_case；`kind` 是单个 lowercase 字母数字原子。多段 kind、大写、连字符、连续下划线和 kind 中的下划线均不合法。

`x` 是架构文件标识。业务 subject 可以与 namespace 同名，是否为架构文件只由 `x.` 前缀决定。

示例：

```text
internal/handler/device.handler.go
internal/handler/device.dto.go
internal/handler/x.shared.dto.go
internal/handler/x.http.endpoint.go

internal/service/device.service.go
internal/service/device.commands.go
internal/service/device.types.go
internal/service/x.shared.support.go

internal/repository/device.schema.go
internal/repository/device.store.go
internal/repository/x.shared.support.go
internal/repository/x.store.repository.go
```

## Subject Ownership

三个层分别是单一 Go package，因此 Go 编译器不会隔离同一层中的不同 subject。默认规则使用文件名和声明名补足这一边界。

业务文件中的 subject-specific 导出声明必须以当前 subject 的 UpperCamel 名称开头：

```text
device.commands.go  DeviceCreateRequest
device.dto.go       DeviceDetailDTO
device.types.go     DeviceStatus
device.mapper.go    ToDeviceRow
```

`support` 和 `types` 中的导出 const 使用 `{Subject}*`；错误 var 使用 `Err{Subject}*`。`support` 中导出的错误 helper 使用 `Wrap{Subject}*`、`Is{Subject}*` 或 `As{Subject}*`。私有 helper 继续由文件角色策略约束，不要求 subject 前缀。

无法归属于单个 subject 的导出声明应放入对应架构文件，例如：

```text
x.shared.types.go
x.shared.support.go
x.shared.mapper.go
```

业务 subject 还会从声明名推断规范 snake_case。例如 `DeviceGroupService` 对应 `device_group.service.go`。

## 文件角色

默认允许的文件 kind：

```text
handler:    free, support, types, mapper, context, dto, endpoint, handler, middleware
service:    free, support, types, mapper, commands, provider, service
repository: free, support, types, repository, schema, store
```

每个 kind 都显式绑定声明策略。自定义 kind 如果只需要命名和依赖约束，必须显式映射到 `free` policy；未知 policy 不会隐式放行。

默认架构 namespace：

```text
handler:    shared, http
service:    shared
repository: shared, store
```

主要声明契约：

```text
*.support.go      类型、const、Err* var、support helper；导出声明服从 subject ownership
*.types.go        类型、const、Err* var，以及错误类型的 Error/Unwrap 方法
*.mapper.go       只能声明包级 To* 函数；业务文件使用 To{Subject}*；禁止 context/database/sql/Bun/GORM import
*.service.go      声明一个 {Subject}Service、New{Subject}Service 和对应 receiver 方法
*.commands.go     只能声明 Request/Response/Result/Item 类型；导出类型使用 {Subject}* 前缀
*.provider.go     声明 {Subject}Provider、New* 构造和对应 receiver 方法

*.handler.go      声明私有 {subject}Handler、Register* 函数和对应 receiver 方法
*.dto.go          只能声明 DTO/DTOs 类型；导出类型使用 {Subject}* 前缀；不能声明函数
*.context.go      仅 x.http；声明私有 *Key 类型和 Context* / *FromContext helper
*.endpoint.go     仅 x.http；必须声明 Endpoint struct 和 NewEndpoint
*.middleware.go   仅 x.http；声明 Middleware、Endpoint receiver 方法和 helper
*.repository.go   仅 x.store.repository.go；必须声明 Store struct 和 NewStore
*.schema.go       声明 {Subject}*Model struct、以 subject 开头的 schema 生命周期函数和 *Model receiver hook
*.store.go        只能声明当前 subject 的 Store receiver 方法
*.free.go         声明内容不受限制，但产生 warning 并进入 free 文件审计清单
```

默认配置不再提供 `utils` 文件 kind。需要迁移遗留 helper 时，应将函数移动到同 subject 的
`.support.go`，或在项目自定义配置中显式注册 `utils` policy。

### Store 方法

导出的 store 方法使用以下动作前缀：

```text
List Fetch Count Exists Create Update Delete Upsert Add Remove Replace
```

动作后必须包含当前文件 subject，方法名不得暴露 `Where`、`Query` 或 `SQL` 等查询实现细节。所有 Store 方法都必须：

```text
第一个参数是 context.Context
最后一个返回值是 error
返回值形状与 List/Fetch/Count/Exists 等动作匹配（风格检查）
```

`StoreMethodActions` 可扩展领域动作。未配置的动作以及 CRUD 返回形状不匹配只产生
`filelayout/store-style` warning；subject 归属、context、error 和查询实现细节仍是 error。

### 持久化边界

```text
service 层所有非 `free` 文件不得直接依赖 database/sql、gorm、bun、pgx、mongo、firestore、dynamodb 等 API
service 文件不得引用 repository.*Model
service 公共方法、公共函数和导出 contract 类型不得暴露 repository 的 Model、Row、Patch、Create、Filter 类型
service 层的 provider/support/types 不得声明 Bun 持久化 tag
repository support/types 不得声明 *Model 或 Bun/GORM tag；schema model 放在 .schema.go
```

内部实现和 service 私有方法可以使用 repository 的非 Model 类型；对外签名边界单独检查。

`MaxSupportDeclarationLines > 0` 时，规则累计 support 文件中 type、const、var 的 AST 声明行跨度。声明范围内部的空白行计入；import、函数以及声明之间的空白行不计入。超过阈值时应拆到同一 subject 或 namespace 的 `.types.go`。

## Subject Anchor

默认将 service 层的 `service` kind 配置为 subject anchor。同一业务 subject 如果声明了 `commands`、`provider`、`support` 或其他受检文件，也必须存在：

```text
{subject}.service.go
```

该文件必须继续满足 `{Subject}Service` 和 `New{Subject}Service` 的声明契约。架构 namespace 和 `free` 文件不参与 anchor 检查。

自定义 qualified-kind 层可以配置自己的 anchor；package-kind 层没有文件级 subject，不能配置 anchor。

## 分层依赖

默认禁止的 import：

```text
handler    -> repository
service    -> handler
repository -> handler
repository -> service
```

因此默认正向依赖是：

```text
handler -> service -> repository
```

`layerdeps` 检查当前 module 中、能够映射到 analyzed root 和 `DependencyLayerDirs` 的 import。它不检查第三方 module。
filelayout 中 provider、support 等角色的跨层 import 也通过 package graph 判断，不依赖固定的
`/internal/service` 字符串路径。

`free` 文件仍参与分层依赖检查。未落在 `LayerDirs` 中的非测试 Go 文件不会被 filelayout 强制改名，但会通过 `Analysis.UnclassifiedFiles` 和生成文档列出。
设置 `WarnUnclassifiedFiles=true` 后，这些文件还会产生 warning，但不会使分析失败。

可选审计项：

```text
WarnSubjectPrivateAccess   报告同层不同 subject 间的私有声明访问
MaxSharedDeclarationLines  限制 x.shared 文件声明规模；0 表示关闭
```

## 例外与审计

### Free 文件

`free` 是显式的结构豁免，不是普通业务 kind：

```text
跳过声明内容检查
跳过 subject ownership 和 namespace 白名单
保留文件名语法和 kind 检查
保留 layerdeps 检查
产生 filelayout/free-file warning
进入 Analysis.FreeFiles、JSON 和 Markdown 审计清单
```

warning 不会使 `Analysis.Passed()` 失败，但会持续暴露尚未归类的结构债务。

### 未分类文件

未落在 `LayerDirs` 中的文件进入 `Analysis.UnclassifiedFiles`。该清单只用于审计，不自动产生 error。

## 架构索引

架构索引是只读分析结果，不参与规则是否通过的判定。它复用同一次静态扫描，不连接数据库，也不执行业务代码。

```text
handler      *.handler.go 中的 *Handler、Register* 和 receiver 方法
service      *.service.go 中的 *Service、New*Service 和 receiver 方法
store        *.store.go 中的 Store receiver 方法
repository   *.schema.go 中的 *Model、Bun table tag、字段 tag 和 ForeignKey 字符串
```

架构索引不表达 API endpoint。API 契约应由项目使用的 API 框架或 OpenAPI 工具生成。

JSON 分析 schema 当前为 v3。实体使用统一的 `identity`：

```json
{
  "identity": {
    "layer": "service",
    "subject": "device",
    "kind": "service"
  }
}
```

架构文件使用 `namespace` 字段。free 文件不进入实体索引，而是进入独立审计清单。

## 诊断契约

每条结构化诊断包含稳定的 `ruleId`、细分 `code`、`severity` 和源码范围。例如：

```text
filelayout/invalid-filename
filelayout/subject-ownership
filelayout/missing-subject-anchor
filelayout/free-file
layerdeps/forbidden-import
```

可机械迁移的问题可以包含 `suggestedFix`；旧式 `x_http.*.go` 命名会提供结构化 rename fix。

推荐通过单元测试同时检查规则和文档漂移：

```bash
go test -count=1 ./internal/testutil/tddcheck
go test -count=1 ./internal/testutil/tddcheck -update
```

## 自定义配置

配置字段：

```text
IncludeTests               加载测试变体和 *_test.go
BuildFlags                 传给 go/packages 的 build flags
StrictPackages             package error 是否在加载阶段直接失败
LayerDirs                  参与 filelayout 的层目录名
DependencyLayerDirs        参与 layerdeps 的层目录名；nil 时等于 LayerDirs
SkipDirs                   扫描时跳过的目录名
LayerRules                 禁止的 import 关系
LayerFileNameModes         每层使用 qualified_kind 或 package_kind
LayerKindPolicies          每层允许的 kind 及声明 policy ID
LayerSubjectAnchorKinds    qualified-kind 层的 subject anchor
ArchitectureNamespaces     每层允许的架构 namespace
EscapedSubjectSuffixes     不允许编码进业务 subject 的 kind 或动作词
ForbiddenWeakSubjects      不允许使用的弱 subject
PublicTypeBoundarySuffixes service 公共签名禁止暴露的 repository 类型后缀
MaxSupportDeclarationLines support 声明的最大累计 AST 行跨度；0 表示关闭
StoreMethodActions         Store 可识别动作；未知动作和返回形状只产生 style warning
WarnSubjectPrivateAccess   同层跨 subject 私有声明访问 warning
WarnUnclassifiedFiles      未分类文件 warning
MaxSharedDeclarationLines  x.shared 声明最大 AST 行跨度；0 表示关闭
LayerPackageNames          严格层目录对应的 package 名
SubjectOwnershipModes      按 layer/kind 设置 subject ownership 为 error、warning 或 off
Initialisms                自定义 snake_case subject 到 UpperCamel 的缩写映射
FailOnWarnings             将 warning 诊断视为 Analysis.Passed 失败
```

slice 或 map 字段为 `nil` 时继承对应默认值。显式空集合用于关闭可选项；`LayerFileNameModes`、`LayerKindPolicies` 等必需的逐层配置仍必须覆盖所有 `LayerDirs`。

默认三包模式应继续使用 `qualified_kind`。`package_kind` 主要用于目录本身已经表达 subject 的自定义层，不改变默认三包结构。

`LayerDependencyRule` 可以通过 `SourceRelPrefix`、`TargetRelPrefix` 及其 exception 字段收窄规则作用范围。所有前缀都相对 analyzed root。

默认 subject ownership 是 error。需要逐步迁移某个文件角色时，可使用：

```json
{
  "subjectOwnershipModes": {
    "handler": {"dto": "warning"}
  },
  "initialisms": {
    "rbac": "RBAC",
    "uuid": "UUID"
  },
  "failOnWarnings": true
}
```

subject ownership 和 subject inference 诊断会尽可能提供源码替换或文件重命名建议；这些建议通过
`Diagnostic.SuggestedFix` 的 `edits` 和 `rename` 字段提供。
