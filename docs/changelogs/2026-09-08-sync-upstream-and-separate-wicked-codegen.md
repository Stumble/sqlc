# refactor: sync upstream and separate wicked code generation

> Publication update (2026-09-09): the user requested a clean upstream-based PR
> containing exactly two commits: core changes, then the wicked backend. The
> earlier merge-based plan and commit references below are historical; the final
> publication structure is recorded at the end of this document.

## 1. Background and Current State

### Objective and agreed scope

将 Stumble/sqlc 同步到现代上游，并重新划分编译器与 wicked Go backend 的职责。
用户已明确选择：需要改动 sqlc 前端的能力继续在 fork 中维护；其余定制尽量归入生成后端。
不再以“完全不改上游”为约束，也不为此额外引入配置 wrapper、第二套 SQL parser 或类型分析器。
所有补丁，包括准备贡献上游的通用编译器修复，都先以 commit 形式保存在自己的 fork。
先完成 fork 的实现及整体验证，确认自己的整套流程可用后，再整理通用补丁向上游提交 PR。
上游是否接收不阻塞本次迁移。

目标仍是保持现有下游的安装、配置、SQL 和业务调用习惯。以下保留批准的架构和实现设计，
实施结果、验证证据和已知限制见第 7-8 节。

### Verified implementation and history

- 当前 wicked 基线：`v2.3.4` / `88a55112526b360dcf46ccdefe9e586e57106e27`。
- 与上游共同祖先：`e6c7cb31f7f218874ce5dcf883964d9e7a5328cd`，2023-09-20。
- fork 相对共同祖先有 23 个独有提交，净修改 45 个文件。
- 实验使用官方稳定版 `v1.31.1`；也检查了上游 main `3c2546a4b` 的接口与近期架构变化。
  最终同步目标版本属于后续设计需要明确的版本选择；本文不把 main 与稳定版视为相同代码。
- `d431dc897`：自定义注释、wpgx 生成器、主模型选择、schema 逆序加载、Schema/Load/Dump 等初始实现。
- `a2d6677ef`、`dad80f45f`：CountIntent 统计以及 SELECT 默认值。
- `35a0729e3`：副本支持、ReadOnlyQueries、allow_replica。
- `8993e7966`：按参数引用上下文选择类型推导位置。
- `45952f709`：wpgx JSON/JSONB 映射到 json.RawMessage。
- `ae8e08c80`、`88a551125`：并发失效的数据竞争修复、nil cache 保护。

当前前端会解析 `-- -- key: value`，把结果放入 Query.options，并直接依赖 Go generator 的选项常量。
它还反转展开后的 schema 文件列表，以原来第一份 schema 标记主模型，并保存其原始 SQL。
Go generator 根据这些信息生成 wpgx/dcache 调用、类型映射、失效参数、Schema/Load/Dump 和副本 API。

### Diagnostic evidence

在隔离目录使用官方 v1.31.1 和仅捕获请求的 process plugin：

- bookstore 保留原 schema 顺序时，revenues 的物化视图报 `relation "books" does not exist`。
- 仅在实验配置中切换生成入口、反转依赖顺序后，5 个包、41 条查询全部通过编译。
- 41 条查询的自定义注释全部进入标准 Query.comments；无需为注释选项新增协议字段。
- 标准请求包含分析后的表、列、参数、SQL 和注释，但没有旧 fork 的主模型标记、原始 schema SQL 或完整 AST。
- `sqlc.narg('id') IS NULL OR id = sqlc.narg('id')` 得到 any。
  只在第二次引用加 bigint cast 仍为 any；第一次引用加 cast，或先出现列比较，则得到 nullable int8。

这些是编译及请求传递实验，不是 wicked backend 迁移、数据库执行或完整 E2E 的验证。
bookstore 当前快照为 `20cb452`，停在 2024 年，不能单独证明后续 wicked 修复已被保留。

### Terminology and known drift

- 已确认的“主”规范是第一份 schema 与主模型选择。
- 数据库 PRIMARY KEY、主模型和 cache key 是不同概念。现有 wicked 专有实现中没有发现额外的主键推导器；
  Dump 目前按可排序 Go 字段选择排序列，源码仍把索引信息列为将来可能使用的输入。
  用户提到的“主 key”暂不据此扩展出新的主键功能，后续以具体既有代码为范围。
- 旧副本检查只判断顶层 SelectStmt，不能证明无写入 CTE、行锁或函数副作用。
- GUIDE 部分版本与类型描述早于当前代码；兼容行为以当前 v2.3.4 实现和真实调用为依据。

## 2. Problem Model and End-to-End Behavior

### Causal problem

多年未同步让上游编译器改进与 wicked 特性混在同一旧实现里。
旧的编译器还承担生成选项的解释和默认策略，使通用 SQL 分析与特定 Go backend 相互依赖。
此次目标是恢复上游演进能力，同时明确哪些定制应随编译器维护，哪些由 wicked generator 独立维护。

### Behaviors and compatibility

- **B1 — Existing entrypoint:** 下游继续使用 wicked-sqlc 的发布与安装方式，以及现有 `sqlc generate`、
  `sqlc diff` 和 `gen.go.sql_package: wpgx` 配置习惯。后端的逻辑独立不要求用户另外安装工具。
- **B2 — Schema conventions:** 保留现有主 schema、主模型、依赖 schema、分区和物化视图的已实现约定。
  schema 加载和对象身份由前端处理，backend 消费明确的主模型与源 schema 信息。
- **B3 — Query options:** 保留已有注释语法。backend 从标准 comments 解析 timeout/cache/invalidate/
  count_intent/allow_replica，并负责这些生成选项的格式、默认值和组合校验。
- **B4 — Compiler facts:** 前端负责 SQL 解析和类型推导，同时提供生成所需的语句结构事实。
  语句类型与 `:one/:many` 等结果基数注解保持区分；结构事实不被描述为完整的只读证明。
- **B5 — Generated contracts:** 保留现有 Go API、nullable 与 JSON 类型约定、cache key、失效参数、
  :one 无结果行为、Schema/Load/Dump、WithTx 和 UseReplica 等调用契约。
  生成文本的排版与内部组织不作为逐字节兼容承诺。
- **B6 — Runtime semantics:** 迁移须保留 timeout、缓存命中/回源、无 cache、事务提交后的失效、回滚、
  多失效目标以及副本方法的现有语义；2025/2026 年修复属于兼容基线。
- **B7 — General fixes:** 对已证实需要保留的参数推导等通用修复，在 fork 中实现可独立测试的补丁。
  这些修复先提交到 fork，与 wicked 专属修改一同完成整体验证，再进入上游 PR 阶段。
  不要求用户为避免维护 fork 而批量改写 SQL，也不把类型推导转移到 backend。

### Failure and degraded behavior

- **F1 — Invalid schema:** 主模型选择有歧义或 schema 依赖不可解析时，生成应失败并定位到输入，不能静默选错表。
- **F2 — Invalid options:** 缺失必需 timeout、无效时长、未知选项、无效失效目标等继续得到明确的生成错误。
- **F3 — Inference regression:** 上游更新导致参数/返回类型退化或已有 SQL 无法编译时，作为具体迁移回归处理。
  backend 不通过猜测 Go 类型掩盖分析错误；既有合法动态类型不能被一概判错。
- **F4 — Replica policy:** 单独讨论是否加强对写入 CTE、行锁和函数副作用的限制；
  不能把更严格策略偷偷混入“行为无感”的迁移中。
- **F5 — Missing metadata:** wicked backend 缺少必要主模型或语句信息时明确拒绝生成，不靠重新猜测源文件归属补救。

### Non-goals and authority

首轮不附带升级 wpgx/dcache 的运行时设计、不新增主键相关产品能力、不重写通用 SQL 分析器，
也不以零 upstream diff 为验收指标。贡献上游属于 fork 完成整体验证之后的阶段；
在此之前不提交上游 PR，也不将通用修复仅保存在上游分支而遗漏自己的 fork。
仓库发布、生产部署和下游批量更新仍是各自的后续工作，本文不报告它们已完成。

## 3. Research, Findings, and Architecture Decision

### Alternatives discussed

| Direction | Consequence | Disposition |
|-----------|-------------|-------------|
| 继续在原 Go generator 上叠加全部定制并整体合并上游 | 保留熟悉的结构，但前端和生成策略继续耦合，合并与回归边界较差 | 不作为本次目标架构 |
| 不改上游，增加配置 wrapper，并在 backend 重新解析 SQL | 可追求原版依赖，但增加配置转换、源文件传递和重复解析，类型推导差异仍需处理 | 用户已明确放弃零修改约束 |
| 维护必要的 sqlc fork 改动，独立维护 wicked backend | 前端保留必要语义与通用修复，生成策略归后端，按职责同步上游 | 用户选择的方向 |

### Selected responsibilities

| Responsibility | Owner |
|----------------|-------|
| SQL grammar, catalog, column/parameter inference | Upstream compiler plus necessary generic fixes |
| Wicked schema load order, primary schema/model identification, source preservation | Wicked-specific frontend integration in the fork |
| Statement structure facts needed by generation | Frontend, using the already available AST |
| Transport of comments, catalog, types, and necessary metadata | Explicit compiler/backend contract |
| Comment option parsing, defaults, and generation-policy validation | Wicked backend |
| PostgreSQL-to-Go mapping, templates, cache keys, timeout and invalidation code | Wicked backend |
| ReadOnlyQueries / UseReplica / CountIntent generation policies | Wicked backend, consuming frontend facts |
| Whole-config conventions such as package-name uniqueness | Wicked configuration validation at the generation entrypoint |
| Existing command/config selection and packaging | Fork entrypoint and release integration |

### Decisions

- **D1 — Fork where useful:** 必要的前端修改直接在 fork 中完成。改动位置以职责和信息可得性决定，
  不以差异行数为唯一目标，不为绕开前端修改而引入另一套 parser。
- **D2 — Separate generic patches:** 类型推导等通用修复与 wicked 规范分开维护，具备独立的行为回归证据。
  通用修复也先形成自己 fork 内可独立审阅和提取的 commits。
  旧补丁的行为诉求需要保留，旧评分算法并不因此自动成为最终实现或已达到上游接受标准。
- **D3 — Independent generator:** wicked generator 使用独立的生成边界，避免把其规则继续散落进上游默认 Go generator。
  保持一个面向用户的工具分发；初期内置 backend 或插件的具体承载形式在实现设计中确定。
- **D4 — Facts across the boundary:** 前端传递主模型、必要源信息和语句结构；backend 解释选项并决定生成策略。
  消除当前编译器对 wicked Go 选项常量的依赖。字段和编码方式在实现设计中明确，允许必要的协议扩展。
- **D5 — Reuse comments:** 自定义选项复用标准 Query.comments，不继续保留仅为了传递这些选项而增加的前端解析与 Query.options。
- **D6 — Scoped conventions:** wicked schema 和生成约定在 wicked 路径中生效，尽量保持上游默认路径的语义与测试可维护。
- **D7 — Compatibility-led migration:** 以 v2.3.4 的真实调用、生成契约与运行行为为基线，bookstore 为现有样例证据之一。
  单独识别上游新能力、必要兼容修复和有意行为变化，避免无意丢失旧修复。
- **D8 — Fork first, upstream afterward:** 固定顺序为 fork 内实现并提交全部补丁 → 完成生成、回归和下游整体验证
  → 整理可通用的补丁并向上游提交 PR。单个补丁测试通过不等于 fork 整体验证完成；
  上游贡献不与尚未验证完成的迁移并行进行，也不作为自己 fork 可用的前置条件。

### Risks and unresolved design details

- **R1 — Version baseline:** 实验覆盖 v1.31.1，不代表已验证当前 main。同步目标版本及其新增编译路径需在后续设计中明确。
- **R2 — Protocol boundary:** 上游标准协议没有主模型标记和完整语句信息。必要字段、兼容方式和 backend 的承载形式尚未设计。
- **R3 — Replica semantics:** 顶层 SELECT 不等于完整只读或业务允许副本。首轮兼容范围与额外修正要明确区分。
- **R4 — Inference correctness:** 参数上下文评分只是旧实现，最终补丁需要可重复、稳定且有行为覆盖的推导规则。
- **R5 — Consumer coverage:** bookstore 停留在 2024 年，仍需对后续 API/类型/失效修复和实际下游使用面补齐验证。
- **R6 — Key terminology:** 当前确认的是主 schema/主模型与既有 cache key；额外数据库主键规则如确有需求，须先定位具体实现。
- **R7 — Validation status:** 此文完成架构记录，不代表实现、生成产物对比或数据库 E2E 已通过。

### Primary references

- [Wicked guide](https://github.com/Stumble/sqlc/blob/88a55112526b360dcf46ccdefe9e586e57106e27/GUIDE.md)
- [Original schema conventions](https://github.com/Stumble/sqlc/blob/88a55112526b360dcf46ccdefe9e586e57106e27/internal/compiler/compile.go)
- [Query classification and inference changes](https://github.com/Stumble/sqlc/blob/88a55112526b360dcf46ccdefe9e586e57106e27/internal/compiler/parse.go)
- [Generation policy and invalidation wiring](https://github.com/Stumble/sqlc/blob/88a55112526b360dcf46ccdefe9e586e57106e27/internal/codegen/golang/result.go)
- [Runtime templates](https://github.com/Stumble/sqlc/tree/88a55112526b360dcf46ccdefe9e586e57106e27/internal/codegen/golang/templates/wpgx)
- [Bookstore](https://github.com/Stumble/bookstore/tree/20cb452)
- [Standard plugin protocol](https://github.com/sqlc-dev/sqlc/blob/v1.31.1/protos/plugin/codegen.proto)
- [Upstream generator dispatch](https://github.com/sqlc-dev/sqlc/blob/v1.31.1/internal/cmd/generate.go)
- [Official Go plugin extraction](https://github.com/sqlc-dev/sqlc-gen-go)
- [Plugin documentation](https://docs.sqlc.dev/en/v1.31.1/guides/plugins.html)
- [PostgreSQL hot standby restrictions](https://www.postgresql.org/docs/current/hot-standby.html)

## 4. Implementation Design

### 4.1 Baseline, repository scope, and integration

实现方案默认选上游稳定版 **v1.31.1**，保留当前 **v2.3.4** 为 wicked 行为基线。
当前 main 包含额外分析器与方言改造，暂不并入本轮；用户如选择 main，需先更新本节及相关验证设计。
首版将独立 backend 编译进现有 sqlc 二进制，不引入另外安装的 process/WASM 插件。

- 主要仓库：Stumble/sqlc。本地工作目录 `/home/forge/sqlc`，分支 `refactor/upstream-sync`，起点为原 fork main。
- 测试样例仓库：Stumble/bookstore。只调整测试环境、回归用例、必要测试输入与生成样例；不改变业务实现来适配错误的生成 API。
  开始修改时增加同名本地 changelog，引用本记录。
- 上游代码通过正常 merge 纳入，保留 fork 历史和旧标签；不改写 main、不强推、不执行伪合并。
  合并冲突以 v1.31.1 的结构为基础解决，再按下面的职责移植 wicked 行为。
- 通用推导修复与 wicked 专用代码形成可独立审阅的 commits。最终所有必要补丁都必须存在于自己的 fork。
- 不修改生产服务、数据库 schema/migrations、运行环境或当前其他工作区的下游生成文件。

已完成的规划验证：merge-tree 预检报告 20 个冲突文件，没有执行工作树合并。
v2.3.4 源码已构建成隔离基线二进制，并在 bookstore 的隔离快照重生成成功；
重生成后的全部 Go 包及测试编译通过（仅编译，未连接数据库执行）。
相对 bookstore 保存的 v2.3.0 产物，有 16 个文件变化，包括 json.RawMessage 和失效修复。
后续差分以这份重新生成的 v2.3.4 产物为基线，避免把历史差异计入本次迁移。

### 4.2 Exact change map

| Location | Responsibility / intended change |
|----------|----------------------------------|
| `internal/config/config.go`, `validate.go` | 识别现有 wpgx 配置；wicked 范围内的有效 package 名唯一性校验；复用上游其他配置验证 |
| `internal/compiler/compile.go`, `engine.go`, `result.go`, new `wicked.go` | wicked schema 逆序加载、主 schema/主对象识别与保存；普通路径继续使用上游行为 |
| `internal/compiler/parse.go` | 通用参数引用选择修复；删除旧版生成选项解析/策略依赖 |
| `protos/plugin/codegen.proto`, generated `internal/plugin/*` | 增加隔离的 wicked 元数据契约，保留标准 comments；使用生成器更新绑定 |
| `internal/cmd/shim.go` | 将编译结果、主对象和 AST 结构事实转换为生成请求 |
| `internal/cmd/generate.go` | wpgx 配置选择独立 wicked handler；保留标准 Go/JSON/process/WASM dispatch |
| new `internal/codegen/wicked/` | 独立 Go backend，基于现代生成接口移植 wicked 类型/参数模型、选项、模板、imports、cache key 与 Dump/Load |
| `internal/codegen/golang/` | 回到上游默认 generator 实现，移除夹杂其中的 wicked 分支和旧专用文件 |
| `internal/compiler/*_test.go`, `internal/cmd/*_test.go`, `internal/codegen/wicked/*_test.go` | 编译事实、协议、生成行为与选项错误的测试 |
| `internal/endtoend/testdata/wicked_*` | CLI 生成/诊断 fixtures；正确期望由检查后的 baseline 和具体行为断言确定 |
| new `scripts/test-local/` | 为上游测试启动本次专属 PostgreSQL/MySQL，向测试进程传入 URI，并按容器 ID 清理 |
| `Makefile`, `go.mod`, `go.sum`, CI workflows, `GUIDE.md` | 新版工具链、单二进制构建安装、可重复 proto 生成、测试入口、最新使用与贡献说明 |
| bookstore `pkg/usecases/*_test.go`, new `internal/testenv/`, `Makefile`, CI | 专属 PostgreSQL/Redis 环境、事务/缓存/timeout/副本运行回归 |
| bookstore `pkg/repos/*/*.go`, affected goldens | 从新工具生成并审阅，与当前 fork baseline 对照；不手改生成代码 |

领域参考中的数据库安全、serde 和测试隔离原则适用；这是 CLI/compiler 项目，
不引入 Alva 服务端 testsuite、网关鉴权或完整 alva-local-dev 服务栈。
“业务服务不直接测试生成 repos”的惯例不适用于生成器自身的输出验证。

### 4.3 Compiler/backend contract

标准 GenerateRequest / GenerateResponse、Query.comments、catalog 和 SQL 类型字段继续使用上游结构。
专用信息集中到一个契约内，不增加一套通用自定义参数协议。

拟定的核心 proto 增量（现有字段省略）：

```proto
message GenerateRequest {
  oneof backend_metadata {
    WickedMetadata wicked = 1000;
  }
}

message WickedMetadata {
  string primary_schema_path = 1;
  string primary_schema_sql = 2;
  Identifier primary_relation = 3;
  map<string, bool> query_is_select = 4;
}
```

- `primary_relation` 使用 catalog/schema/name 身份，不能只用表名猜测；视图也使用 relation 身份。
- `query_is_select` 的 key 是同一 query set 内已校验唯一的查询名；每条具名查询必须有记录。
  false 与缺少 key 不同，缺少记录是明确的接口错误。
- 该 bool 精确表达旧代码的“顶层 SelectStmt”，不命名为 read_only，也不新增未使用的行锁/副作用分析。
- 只有 wicked 模式附带元数据。oneof 用于保留普通请求的 JSON 形状：上游 JSON generator 使用
  EmitUnpopulated，而未设置的 oneof 不被输出；该兼容点必须有测试。
- 1000 是本 fork 使用的增量字段号，后续同步仍须核对冲突，不假定高编号永不被上游占用。
- 旧 fork 删除的 Query.options 9、Catalog.raw_sqls 5、Table.generate_model 4 保留 reserved tag/name，防止误复用。
  旧 fork 的内部 proto 不是下游应用 API，本次由新结构替换，不承诺旧自定义插件二进制可直接加载。
- 不传完整 AST，不在 backend 依赖 compiler、catalog 的内部 Go 对象或重新读取 SQL 文件。

backend 的入口与上游一致：

```go
func Generate(ctx context.Context, req *plugin.GenerateRequest) (*plugin.GenerateResponse, error)
```

Go 配置走标准 PluginOptions / GlobalOptions。backend 局部适配 legacy wpgx 名称后复用 Go options 的解析能力，
其中覆盖规则/rename 的优先级须通过 fixture 与 v2.3.4 对照。普通 Go generator 不增加 wpgx 校验例外。
生成阶段只返回输出文件或错误，不直接写磁盘，也不修改传入请求，避免多个输出消费者相互影响。

### 4.4 Schema and query pipeline

代表流程是 bookstore 的 revenues query set：

1. 配置识别为 wicked，使用上游 sqlpath 展开 schema 路径；在改变顺序前保留原第一份文件的身份。
2. 对 wicked 路径反转展开后的文件列表，先解析 books，再解析 orders，最后解析 revenues。
   普通上游路径保持原顺序。
3. 用现有 parser/catalog 构建对象；在读取每份文件时应用旧的一文件一个逻辑布局约定，
   物理分区的无新列布局不会被当作第二个主模型。
4. 在原第一份文件中识别主对象，保留用于生成 Schema 的源内容；依赖对象仍留在 catalog 参与查询分析。
5. 类型分析完成后，cmd shim 从 Query.RawStmt 记录顶层 SELECT 与否，并把主对象信息写入 WickedMetadata。
6. backend 仅为主对象输出主模型，保留原来的 enum 与结果结构处理；用传入的源内容生成 Schema、Load、Dump。

优先在 compiler.Result 的私有 wicked 数据中保留主对象身份，避免给所有 catalog.Update 调用添加生成策略参数。
对原始导出 Schema 与用于分析的预处理 SQL 分开处理：保留 v2.3.4 的导出语义，并使用上游解析所需的 migration/psql 预处理。
空 schema、多个主布局、无法匹配 catalog 对象必须报错；错误包含配置/文件及对象上下文。

当前源码的普通 ViewStmt 没有旧 GenerateModel 标记，不能仅凭 GUIDE 就声称已有完整支持。
基线 fixtures 将记录普通视图、物化视图与分区的实际差别；首轮不把新增普通视图能力作为隐藏目标。
新的实验性 database-only 分析如无法提供可靠主模型，wicked 路径须明确诊断，不输出缺失模型的代码。

### 4.5 Generic inference patch

移植现有“同一参数使用信息更充分的引用上下文”修复，不在迁移中设计第二套类型分析器。

- 按参数编号聚合已有 paramRef，保留首次出现顺序；选择原 wicked 评分较高的引用，同分保持首次引用。
- 已有评分语义：显式 cast/比较 100；算术/连接/LIKE/limit/offset 90；BETWEEN 75；IN 70；
  结果目标及 AND/OR 60；一般表达式/NOT 50；函数/其他布尔 40；空上下文与 NULL 判断低优先级。
  具体分支以旧补丁为迁移依据，并用真实 SQL 推导结果验证，不能用评分表本身作为全部测试断言。
- 初版曾把 INSERT/UPDATE 赋值目标提到 100 来规避重复表引用的歧义；下游审计证实这会无意改变 nullable。
  复核后的修复恢复旧评分 60，在 resolveCatalogRefs 中仅去重重复的无别名 relation。
  同一表跨 CTE/主语句不会再被重复计数；不同 alias 仍保留，真实 self-join 歧义不被掩盖。
- 保留未编号参数的编号分配和 named/narg 的 nullability 信息；之后仍使用上游排序及 resolveCatalogRefs。
- 不用 map 的迭代顺序决定参数顺序。复杂或互相矛盾的 SQL 类型约束继续遵循编译器已有诊断能力，
  该补丁不被描述为完整的约束求解器。
- `IS NULL` 在前、cast 在后、比较在后、多次引用、显式 nullable、单次引用、参数编号与重复执行稳定性为关键覆盖。

这项修复形成独立 commit。先在 fork 验证通用和 wicked 两条生成路径，之后再准备上游 PR；
上游评审需要的进一步算法整理仍应先提交并验证在自己的 fork。

### 4.6 Backend policy and runtime compatibility

注释处理：对每条 comment 去除外围空白；以 `--` 开头的项按首个冒号分割 key/value。
保留旧的重复 key 后者覆盖行为、时长最小 1ms、必需 timeout、失效目标存在且已启用缓存等规则。
无效选项由 backend 返回带 package/query 的错误，CLI 返回失败；不把 wicked 校验重新放进 metadata parser。

`count_intent`、`allow_replica` 默认值及 SELECT 的 invalidate 禁止规则由 backend 使用 query_is_select 决定。
首轮保留现有 SELECT 判断和各模板的调用行为；加强行锁、写入 CTE 或函数副作用限制属于单独的行为变更。

以 v2.3.4 模板和调用契约为标准，保留：

- 主模型字段、nullable enum 包装、JSON RawMessage、UUID/time/numeric/range 等映射；Go 类型选择与 SQL 类型推导分开。
- 原有 query_parameter_limit、参数结构、返回结构、指针和失效参数签名，以及 :one 无结果返回 nil,nil。
- 模型 JSON tags、enum helper 等既有规则；不笼统强制所有 options，以免顺带改变 enum tags 等旧行为。
- cache key 的 package/query 前缀、参数顺序、nil 表达与长 key 哈希；跨版本缓存兼容属于验收项。
- timeout 覆盖缓存读取及数据库调用；失效回调继续捕获原 ctx，保留提交后执行、nil cache 保护和并发局部错误变量。
- 无参数失效、多目标失效、**T 失效参数、copyfrom、UseReplica/AsReadOnly/WithTx/WithCache、Schema/Load/Dump。
- 原来已有的错误返回和日志语义，不在迁移时另改 PostExec 的错误传播契约。

批量 :batch* 在旧 wicked 中仍有未完成实现（模板调用 WGConn 不提供的 SendBatch）。
标准上游 batch 能力继续测试；wicked 未支持的组合给出明确诊断，不把生成无法编译的代码算作支持。
任何已在真实下游工作而未被上述清单覆盖的情况，都以基线与调用证据补入验证范围。

### 4.7 Build, test environment, and delivery

- 初始工具链按上游使用 1.26.2；CI 后续安全核查要求升级到同系列修复版本 1.26.8，
  go.mod 的最低版本、工具链和 CI 安全检查保持一致。主机 GOTOOLCHAIN=auto。
  保留默认 CGO 构建，另验证上游提供的非 CGO 路径。旧 v2.3.4 基线可使用当前已成功的构建方式。
- `make proto` 保留 Buf 生成路径；增加 BUF 参数，允许使用固定版本
  `go run github.com/bufbuild/buf/cmd/buf@v1.72.0`。当前没有 buf/protoc，不能手改生成绑定代替生成。
- 保留 wicked 版本标识、make build/install 的使用方式；普通上游 CI 与 fork 的 wicked 回归检查均要保留。
- 上游内置 Docker 测试 helper 使用固定容器名和 5432/3306，不能在共享主机直接依赖自动启动路径。
  `scripts/test-local` 启动本次专属 postgres:16/mysql:9 容器并使用随机宿主端口，
  以它创建的连接串覆盖测试子进程的 POSTGRESQL_SERVER_URI/MYSQL_SERVER_URI。
  复用上游测试的 URI 接口，保持默认测试代码不变；正常退出、失败和信号退出均按本次容器 ID 清理。
- bookstore 的现有测试硬编码 Redis 6379，且会 FlushAll；wpgx suite 会删除并重建测试库。
  新 `internal/testenv` 为测试拥有独立 PostgreSQL 和 Redis 容器、随机宿主端口，并将显式测试配置传给 suite。
  清理只针对本次创建的容器 ID；不借用当前工作区的数据库/Redis，不执行全局容器清理。
- bookstore 继续固定 wpgx v0.3.1、dcache v0.1.3 的运行时依赖；测试基础设施调整不附带运行时升级。
  副本测试延续同实例的独立连接配置，验证生成 API/路由行为，不声称验证真实复制延迟。
- 生产无需 DDL、数据回填或部署操作。回退为重新使用 v2.3.4 工具和对应已提交产物。
- 完成 fork、样例、文档及回归验证后，再进入自己仓库的发布流程；上游 PR 排在整体验证之后。

### Serial Implementation Checklist

- [x] 保存并审阅 v2.3.4 对照产物，建立生成签名/cache-key/运行行为 fixtures（B1/B5/B6；结构检查与基线）。
- [x] 将 v1.31.1 纳入工作分支，按新接口独立迁移 wicked backend，恢复构建和普通生成路径（B1/D1/D3/D6；沿用上游测试）。
- [x] 定义并生成 WickedMetadata，接入主 schema/主对象和 query_is_select，完成 scoped dispatch（B2/B4/F1/F5；契约与 CLI 测试）。
- [x] 独立移植通用参数引用修复，用真实 SQL 推导回归验证后形成独立 commit（B7/F3/D2/D8；test-first）。
- [x] 完成注释选项、类型/模板、失效签名与 key、helper 的迁移，逐项对照基线（B3/B5/B6/F2/F4；测试与迁移并行）。
- [x] 在 bookstore 增加本地 changelog、隔离 testenv 和运行回归，重生成并审阅样例及必要 goldens（B1/B5/B6；回归 test-first）。
- [x] 更新 GUIDE、构建/安装/proto 和 CI，运行第 5 节检查，记录结果并审阅完整 diff（所有 B/F/R；现有覆盖加新增回归）。
- [x] 完成自己的 fork 全部补丁与整体验证，保留独立通用修复；上游 PR 为验证完成后的独立后续工作（D8）。

## 5. Verification and E2E Design

### 5.1 Required evidence

**E2E Required: yes.** 生成成功和 Go 编译不能证明事务失效、缓存、timeout 和副本调用语义。
这里的完整链路是本地 CLI → 生成 Go → bookstore 调用 → 专属 PostgreSQL/Redis，
不涉及 Alva 网关/服务栈、生产、真实设备或云端数据库。

| Behavior | Evidence / plausible regression detected |
|----------|------------------------------------------|
| B1/D6 | 原 wpgx YAML 无需用户修改；普通 PostgreSQL/MySQL/SQLite fixtures 与插件 tests；发现全局施加 wicked 限制 |
| B2/F1/F5 | 主 schema 在首位、依赖在后、物化视图/分区、多个逻辑表/缺少对象；验证模型身份和 Schema 源文 |
| B3/F2 | comments 选项、重复 key、未知 key、缺 timeout、无效 duration/失效目标；检查明确错误与退出码 |
| B4/F4 | SELECT、WITH SELECT、INSERT/UPDATE/DELETE RETURNING 的事实及方法集；锁/写 CTE 单列旧行为 |
| B5 | 生成 Go AST/API 对照、编译使用点、显式类型与 cache-key 断言；发现类型/参数/标签/key 漂移 |
| B6 | 真实 PG/Redis 的缓存回源、无结果、nil cache、提交/回滚、多目标失效、context 和 replica 用例 |
| B7/F3 | SQL 参数推导 fixtures + 标准生成路径；发现仍为 any、nullable 丢失、编号/顺序不稳定 |
| D4 | protobuf round-trip、缺失 metadata、普通请求无 wicked JSON 字段；发现契约或默认生成接口污染 |
| D8 | commit 历史检查 + 全套验证记录，确认通用修复存在于自己的最终 fork |

### 5.2 Representative compiler and generator tests

下面为关键行为的代表性测试设计，helper 将在测试中使用真实 compiler/request，不模拟推导结果：

```go
func TestWickedNullableParameterContext(t *testing.T) {
    // things.id 是 bigint NOT NULL；narg 显式指定参数可空。
    got := compileFixture(t, "nullable_first.sql")
    p := onlyParameter(t, got, "FindThings")
    require.Equal(t, "int8", canonicalSQLType(p.Column))
    require.False(t, p.Column.NotNull)
    // 同样输入重复生成，并检查按参数编号排列的输出一致。
}

func TestWickedPrimaryRelation(t *testing.T) {
    req := compileWickedFixture(t, "materialized_view_primary")
    require.Equal(t, "by_book_revenues", req.GetWicked().PrimaryRelation.Name)
    generated := generateWicked(t, req)
    require.Equal(t, []string{"ByBookRevenue"}, exportedMainModels(t, generated))
    require.Equal(t, fixturePrimarySQL(t), exportedSchemaValue(t, generated))
}
```

schema/model 名称以 fixture 自己定义的名字为期望，不从被测算法生成期望值。
生成文本 golden 作为辅助证据：所有变更必须人工检查，API 断言解析 Go AST/类型信息，
关键 runtime 行为使用实际调用验证。允许解释清楚的格式/版本注释变化，不用全量接受快照掩盖回归。
普通请求 JSON 比较先规范化 JSON，不能依赖 protojson 的空白格式稳定。

### 5.3 Representative runtime test

以下展示提交/回滚的核心断言；fixture 提供既有 Book 和显式缓存/DB 计数，避免仅靠 TTL 或 sleep 判断失效：

```go
// 在有缓存的 q 上读取并缓存原有 Book。
before, err := q.GetBookByID(ctx, id)
require.NoError(t, err)
require.NotNil(t, before)

abort := errors.New("rollback fixture")
_, err = pool.Transact(ctx, pgx.TxOptions{}, func(ctx context.Context, tx *wpgx.WTx) (any, error) {
    err := q.WithTx(tx).UpdateBookByID(ctx, books.UpdateBookByIDParams{
        ID: id, Description: "changed", Meta: before.Metadata, Price: before.Price,
    }, &id)
    if err != nil { return nil, err }
    return nil, abort
})
require.ErrorIs(t, err, abort)
after, err := q.GetBookByID(ctx, id)
require.NoError(t, err)
require.Equal(t, before.Description, after.Description)
// 额外断言回滚没有触发失效/额外回源；相同更新成功提交后，后续读取必须得到 changed。
```

其他运行覆盖：

- 缓存命中不增加数据库调用；缓存缺失时回源，cache=nil 的读写/失效都正常。
- 事务尚未提交时失效回调未执行；成功提交才失效，回滚保持原缓存与数据库状态。
- 多目标、无参数及 **T 失效参数使用正确 key；-race 覆盖旧的共享 err 竞争。
- 注入阻塞的数据库/缓存路径与 context deadline，验证 timeout 生效；不依赖精确毫秒计时。
- :one 无行得到 nil,nil；数据库错误得到 error；:many 错误路径不会发生类型断言 panic。
- 使用已命名 replica 连接执行允许的方法；生成 API 检查禁止普通写入方法出现在 ReadOnlyQueries。
- 复制、Load/Dump 和 JSON/nullable 数据 round-trip；检查 v2.3.4 已改变的 JSON golden，不能沿用旧 base64 期望。

### 5.4 Commands and prerequisites

以下为实施后必跑命令，当前不是已通过的结果：

```bash
# Stumble/sqlc 工作分支
make proto BUF='go run github.com/bufbuild/buf/cmd/buf@v1.72.0'
make build
go run ./scripts/test-local -- go test -count=1 ./internal/compiler ./internal/config ./internal/cmd ./internal/codegen/wicked/...
go run ./scripts/test-local -- go test -count=1 ./internal/endtoend -run 'TestReplay/base/wicked_'
go run ./scripts/test-local -- go test -count=1 -timeout 20m ./...
make build-endtoend
CGO_ENABLED=0 go build -o bin/sqlc-nocgo ./cmd/sqlc

# Stumble/bookstore，Makefile 增加 SQLC 变量以明确待测二进制
make sqlc SQLC=/home/forge/sqlc/bin/sqlc
make sqlc-verify SQLC=/home/forge/sqlc/bin/sqlc
go test -run '^$' ./...
make test
go test -race -count=1 -p 1 -timeout 10m ./pkg/usecases
make lint-fix
```

完整上游测试通过专属容器的显式 URI 使用其既有测试环境接口，并确认实际数据库用例未被静默跳过。
需要的插件测试工具按上游 CI 固定版本准备；远端下载失败或托管服务不可用应单列，不报告为全部通过。
wicked 生成 fixture 不依赖数据库在线；bookstore E2E 必须实际运行在上述专属容器。
bookstore 的 `make test` 将调用隔离 testenv；不使用现有固定端口的 docker-start/FlushAll 组合访问共享实例。

sqlc 仓库没有 make lint-fix，记录 lint 不可用，不发明替代 lint 命令；
proto 生成、编译、必要的 gofmt 和 CI 既定检查仍需完成。bookstore 有 make lint-fix，执行后排除无关格式变动。
生成和 proto 二次运行要无额外 diff；对所有产品内容变更运行相应新检查，最终 review 后执行完整 E2E。

## 6. Human Decisions and Interaction

- 架构已确认：必要的前端改动保留在 fork，生成策略集中于 wicked backend。
- 所有补丁先形成自己 fork 的 commits；自己的整套工具与下游验证通过后，再向上游提 PR。
- 自定义参数复用 comments；保持现有下游输入、安装习惯和生成 API。
- 用户已批准本代码级计划，并授权连续完成实现、验证、自己的 fork push 和 PR 后汇报。
- 执行基线：上游 v1.31.1、同一二进制内的独立 backend、保留现有副本判断，bookstore 为运行验证样例。
- 通用修复继续先提交在自己的 fork；上游贡献遵守 D8 的整体验证前置条件。

## 7. Outcome and Evidence

### Result and review

本小节记录首次发布时的验证；后续真实下游审计推翻了其中“B5/R5 已充分覆盖”的判断。
最终以本文件末尾的 post-publication follow-up 和新验证结果为准，不把初版 CI 全绿当作全面兼容的证明。

上游 v1.31.1 通过正常 merge 纳入；wicked 生成器位于独立包，按旧 wpgx 配置选择。
前端保留 schema 约定和类型推导，元数据使用批准的 oneof 契约，comments 负责选项传递。
通用参数推导补丁为独立 commit `c7f1ceb23`，没有向上游提前提交 PR。

按行为、架构、测试、数据/兼容、运维和文档检查了本次自有差异；上游导入部分与 v1.31.1 tag 对齐。
检查并修复了以下问题，之后重新执行对应测试与最终完整验证：

- 赋值目标的参数绑定优先级，避免 INSERT … SELECT + CTE 在 managed-db 模式下丢失参数信息。
- 新版 pg_catalog 类型限定与旧 JSON/UUID 等 Go 类型映射，以及 db_type overrides 的匹配。
- Go options 解析保留 Catalog 上下文，防止列 override panic；保持局部 rename 优先和旧 override 顺序。
- 主表身份跟随后续 ALTER/RENAME，再生成最终的规范化标识。
- emit_interface 的返回指针与 invalidate 参数签名；有生成 fixture 和编译检查。
- 版本信息改为可通过原 Makefile 链接参数设置，保留 wicked 标识。

| IDs | Result | Evidence |
|-----|--------|----------|
| B1, D1/D3/D6 | DONE | 原 bookstore YAML、单二进制、普通 Go/JSON/插件测试、CGO/非 CGO 构建 |
| B2, F1/F5 | DONE | 普通表、分区、物化视图、重命名、错误布局、主源保存 tests |
| B3, F2 | DONE | 注释选项、重复键、时长/未知选项/失效目标验证、缺 timeout CLI fixture |
| B4, F4, D4/D5 | DONE | QueryIsSelect、INSERT RETURNING 方法集、proto round-trip、普通 JSON 无额外字段 |
| B5 | DONE | bookstore 42 条查询的 17 个 Go 文件与相同输入的 v2.3.4 产物仅版本注释不同 |
| B6 | DONE | 实际 PostgreSQL/Redis 的缓存、提交/回滚、失效、nil cache、JSON、copyfrom、timeout、replica tests |
| B7, F3, D2/D8 | DONE | 独立 commit、nullable 参数和赋值绑定 tests、原上游对照、完整上游测试 |
| R1/R2/R4/R5/R6/R7 | Resolved for this migration | 已明确基线、契约、算法修复、样例覆盖和主模型范围；没有新增主键产品能力 |
| R3 | Retained limitation | 保持原 SELECT 分类；不声称能证明任意 SQL 完整只读 |

### Final local verification

工作目录 `/home/forge/sqlc`，下列检查均已通过：

- `make build COMMIT_HASH=v2.4.0-dev`。
- `go test -count=1 ./internal/compiler ./internal/cmd ./internal/codegen/wicked ./internal/config`。
- `PATH=/home/forge/sqlc/bin:$PATH go run ./scripts/test-local -- go test -count=1 -timeout 20m ./...`。
- `PATH=/home/forge/sqlc/bin:$PATH go run ./scripts/test-local -- go test -count=1 -tags=examples -timeout 20m ./...`，
  含上游全部数据库/生成测试；最终 internal/endtoend 用时约 148 秒，PG/MySQL 使用本次专属容器。
- `make build-endtoend`，含新增 wicked 主模型/interface fixture。
- `CGO_ENABLED=0 go build -ldflags='-X github.com/sqlc-dev/sqlc/internal/info.Version=v2.4.0-dev-wicked-fork' -o bin/sqlc-nocgo ./cmd/sqlc`。
- `bin/sqlc-nocgo diff -f /home/forge/bookstore/pkg/repos/sqlc.yaml`，通过。
- `make proto BUF='go run github.com/bufbuild/buf/cmd/buf@v1.72.0'`，重复生成无额外差异；Buf lint 通过。
- git diff whitespace 检查与提交范围/暂存区凭据扫描通过。sqlc 没有 make lint-fix，未另造 lint 命令；
  Buf 自身的 schema 检查与 Go 编译/测试照常执行。

Bookstore 的 `make test`、Go 1.25.7 与 Go 1.26.2 的 race suite、`make lint-fix` 和生成 diff 全部通过。
14 个 suite 用例及两个 search 子用例使用真实 PostgreSQL/Redis；详情见其本地 changelog。
三处 JSON golden 的变化属于已有 v2.3.4 RawMessage 行为，已逐个检查。
未运行生产数据库或完整 Alva 服务栈，因为本次明确验证的是 CLI → Go → 数据库/缓存链路。

## 8. Remaining Work

- 已发布 [sqlc #16](https://github.com/Stumble/sqlc/pull/16) 与
  [bookstore #2](https://github.com/Stumble/bookstore/pull/2)。CI 的最终状态以对应 PR 为准；不自动合并或发布 release。
- 在自己的整套验证通过的基础上，独立整理通用类型推导补丁的上游 PR。
- 已保留的限制：顶层 SELECT 分类不等于完整只读证明；ordinary view 不能作为 wicked 主模型；
  wicked batch/execlastid 和 database-only 分析不支持；copyfrom 不实现 cache/invalidate 选项。
- 样例 replica 用例验证 API/连接选择，不模拟物理复制延迟；wpgx/dcache 运行时版本未升级。

### CI security follow-up

首次发布后，GitHub 上的六个平台构建、完整 Go 测试、Buf 和 bookstore 验证均通过。
安全检查发现 v1.31.1 上游依赖中的 GO-2026-6061（grpc）和 GO-2026-5970（x/text）。
本地按实际构建工具链扫描还发现 Go 1.26.2 标准库及 x/net 的修复需求。
修复范围限定为 grpc v1.82.1、x/text v0.39.0、x/net v0.55.0 及其必要传递依赖，
并将工具链升级为 Go 1.26.8，安全检查也固定到该构建版本。
这不改变编译器的 v1.31.1 代码基线或下游 wpgx/dcache 版本；修改后重新进行完整验证。

安全修复后的再次验证全部通过：Go 1.26.8 下的 focused tests、完整 examples/endtoend 套件
（internal/endtoend 约 152 秒）、CGO/非 CGO 构建、生成 diff 和 bookstore race suite。
`go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...` 报告 0 个可达漏洞；
工具另提示一个未被调用到的导入包/模块公告，不属于可达代码路径的失败项。

参考：[Go security advisories](https://pkg.go.dev/vuln/GO-2026-6061)、
[x/text advisory](https://pkg.go.dev/vuln/GO-2026-5970)、
[Go release history](https://go.dev/doc/devel/release#go1.26.8)。

### Post-publication compatibility follow-up

用户要求扩大旧版本对照，随后授权修改当前 PR，并用新 binary 验证 Alva mono repo；允许少量有依据的改进。
这轮使用追加 commits，不重写已推送历史，不修改 Alva 原工作区、业务代码或生产数据库。

审计确认的修复/取舍：

- A1：恢复 google/uuid，增加实际 import/类型身份及 nullable UUID 的编译断言。
- A2：恢复旧参数上下文评分，修复重复无别名表的假歧义；保留 nullable 赋值、narg 及真实 self-join 的含义。
- A3：主模型候选与新布局计数分开；继承表、空 CREATE 后 ALTER、重命名及分区继续保留。
- A4：Docker release 使用 info.Version，与 make build、CLI 和生成文件版本一致；新增实际二进制测试。
- A5：保留 exec + RETURNING 曾导出的 Row 类型；不在迁移中顺带删除公共 Go 类型。
- A6：保留旧 column-only struct tags，避免默认启用 db_type tags 后改变 JSON/共享缓存格式。
- 保留合理改善：接口签名修正、局部变量冲突修正、CopyFrom 原始列名、内建类型识别；不为了逐字一致恢复已证实旧缺陷。

新的 wicked_compat fixture 来自 v2.3.4 的相同 SQL/config 生成结果，覆盖继承模型、Google UUID、Row API 和 JSON tags。
测试对旧 JSON payload 进行实际编解码，CI 编译消费者；不再只比较类型名称字符串。
参数编译器测试先验证旧签名回归及重复 relation 假歧义均失败，再修复；发布测试实际构建并运行二进制。

完整上游测试发现 insert_select_param 之前只在 managed-db 模式下测试：官方 v1.31.1 本地 compile
对合法的 INSERT … SELECT FROM 同表错误报告 name 歧义，数据库 fallback 丢失了静态参数 nullability。
新去重逻辑让本地分析成功，参数成为非空 int64/string；已将该 fixture 扩为 base + managed-db，
使用生成器更新其输出并核对仅移除 pgtype 导入、两个字段类型变化。这是有独立证据的通用修正，
不是为了让失败测试通过而回避差异；Alva 218 组输入与旧 v2.3.4 生成结果仍一致。

执行清单（本次最终证据将补在此处）：

- [x] 修复 A1–A6 并加入针对性回归。
- [x] 重新执行完整上游测试、fixture 编译/编解码和 bookstore 数据库/race 验证。
- [x] 在隔离副本重新生成 mono repo 的全部 wpgx 配置，逐项审阅差异并验证真实消费者（trex 全量本地编译限制见下）。
- [x] 重跑双版本矩阵，归类少量合理差异，复核完整 net diff。

发布沿用 sqlc #16 与 bookstore #2；当前 HEAD 的 CI 和审查反馈状态以 PR 页面为准，不自动合并。

#### Final follow-up evidence

修复提交：`cead568ed`（通用 compiler）与 `62d9e5b26`（wicked/release 兼容性）。
最终 net diff 已按行为、职责边界、测试、数据兼容、构建和文档重新审阅；默认 Go backend 保持 upstream v1.31.1 源码。

sqlc 工作目录 `/home/forge/sqlc`：

- `make build COMMIT_HASH=v2.4.0-dev`，最终二进制从干净 `62d9e5b26` 构建。
- CGO 与非 CGO 的 compiler/cmd/wicked/config `go test -count=1` 均通过。
- `GOMAXPROCS=2 PATH=/home/forge/sqlc/bin:$PATH go run ./scripts/test-local -- go test -p 2 -parallel 8 -count=1 -tags=examples -timeout 20m ./...` 全部通过；endtoend 144.467s，含 base/managed-db、SQLite/MySQL/PostgreSQL、插件及新 fixture。
- `GOTOOLCHAIN=go1.26.8 make build-endtoend` 通过；testdata module 中 `go test -count=1 ./wicked_compat/go` 通过。
- `CGO_ENABLED=0 go build -ldflags='-X github.com/sqlc-dev/sqlc/internal/info.Version=v2.4.0-dev-wicked-fork' -o bin/sqlc-nocgo ./cmd/sqlc` 通过；该 binary 的 bookstore `diff` 通过。
- `GOMEMLIMIT=1GiB GOMAXPROCS=2 go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...`：0 个可达漏洞；另有未调用到的导入包/模块公告。
- 按 PR base 扫描全部新增提交：未发现凭据；本轮没有 proto 修改，协议 round-trip/标准请求测试随全套重新通过。

bookstore 工作目录 `/home/forge/bookstore`：新 binary 的 `make sqlc`、`make sqlc-verify` 和 `git diff --exit-code -- pkg/repos` 均通过，17 个输出无需更新。
`GOFLAGS=-count=1 make test` 通过（8.690s）；Go 1.25.7 race suite 通过（11.851s），Go 1.26.8 race suite 通过（13.768s）；`make lint-fix` 为 0 issues。

消费者在 `/tmp/sqlc-mono-validation.regHRh/{baseline,current}` 的独立副本中验证：

- 扫描覆盖 8 个仓库、226 组 wpgx 配置；689 个本次生成的 Go 文件与同输入 v2.3.4 产物仅版本注释不同。
- alva-backend、jagent、alfs、connectors、llm-data、synthdb、forge 的全量 Go 编译通过；go.work 中的本地模块使用明确的 `./path/...` 一起编译。jagent 按其 AGENTS.md 使用 Clang 21 与 `CGO_CXXFLAGS=-nostdinc++`。
- trex 的 36 个新生成文件与当前已提交文件也仅版本注释不同；`go build -p 1 ./internal/repos/...` 通过。其同一源码提交 `56027c847620d0a16285ed4d1736301ac1980e99` 的 CI、lint 和镜像构建已确认成功。
- **验证限制：** trex 完整本地构建的 CCXT 第三方包超过本环境内存软限制。默认构建及限制优化/并发的诊断尝试均已取消；不计为通过。采用本地生成包编译、源代码等价对照与同源码既有 CI 证据；未修改依赖版本、业务代码或提高宿主机限额。
- 与更老的已提交生成代码相比，差异为旧 fork 已有的 nil-cache/并发失效修复，以及一处手写生成 wrapper 的等价模板化；均已审阅。原 mono repo 工作区及其子模块改动完整保留，没有提交消费者生成文件。

双版本矩阵 147 组：113 组生成成功且仅版本注释不同，23 组为已解释的接口/变量冲突/CopyFrom/内建类型识别改善；另有 3 组旧版可生成但无法编译的 batch、3 组新版修复的关键字输入、5 组两版均拒绝的非法 CopyFrom。
成功生成案例的编译结果已单独分类，没有新引入的“旧包可编译、新包不可编译”；没有把旧版本本来失败的组合计为新回归。

资源诊断记录：并行 CCXT 构建引起 cgroup memory.high 压力，期间 bookstore 短 deadline 与一次上游 20m 测试超时；安全扫描亦主动终止。
停止重型编译后，未修改 SQL timeout、生产代码或测试期望，重新执行得到上述完整通过结果。所有本次测试容器均清理。
未运行完整 Alva 服务栈/生产数据库；这次的运行时 E2E 边界仍是已批准的 CLI → bookstore → 自有 PostgreSQL/Redis。

### Upstream-based two-commit publication (2026-09-09)

用户要求 PR 直接从 upstream 起步，不再把 merge upstream 的历史放进 review。
该决定替代第 4.1 节中的 merge-based publication 方式，不改变已验证的版本或代码行为。

- 准确基线：upstream v1.31.1，`a95e91d70ad9e1181253c333a1cfdd75ae4b95a5`。
- PR 仍在 Stumble/sqlc：base 为新建的 `integration/upstream-v1.31.1`，初始指向上述原始上游提交；
  head 为 `refactor/wicked-on-upstream`。不向 sqlc-dev/sqlc 发 PR。
- 只有两个线性提交，区间中没有 merge commit：
  1. `feat(core): apply wicked compiler and protocol changes to upstream`：编译器、schema/配置约定、
     协议及事实传递、公共版本/构建、安全依赖、核心测试与测试工具。
  2. `feat(wicked): add the compatible wpgx code generation backend`：独立生成器、模板、CLI 注册、
     backend/消费者契约测试、样例 CI、GUIDE 和本迁移记录。
- CLI 注册与依赖 backend 的集成测试位于第二个提交，避免核心提交引用尚未存在的生成器。
- 旧的 `refactor/upstream-sync` 分支及 #16 保留历史；新 PR 替代旧 review，不重写旧分支或 main。
  合并新 PR 更新的是 integration 分支；迁移 main/默认分支需之后明确决定。

结构验证：在补充本段和更新 README/GUIDE 的分支说明之前，重建的 index tree 与已验证的
`aa0ff2cdb` 完全相同（tree `1c3c895c10634e2292636bbe68b17370f673bd61`）。
最终与该版本相比仅说明文档有变化（README/GUIDE/本记录，以及历史 wicked_change_logs 的行尾空格清理），
产品代码、依赖、协议、模板、生成文件和 CI 配置逐字一致。
因此之前的 8 仓库/226 配置兼容性结果及其 trex 资源限制继续适用；不因重排提交而重复高资源消费者构建。
核心提交已独立通过 CLI 构建及 compiler/config/cmd 检查；完整树继续执行构建、focused tests、
生成契约/JSON tests 和 bookstore diff，最终 CI 状态以新 PR 当前 HEAD 为准。
