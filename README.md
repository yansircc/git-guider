# git-guider

Web 闯关式 Git 训练游戏。在浏览器终端里输入真实 git 命令，通关学会 Git。

## 当前状态

核心领域层已实现（Step 0），transport/UI 尚未开始。可以通过测试验证全部功能。

## 快速开始

```bash
# 确保 Go 1.23+ 和 git 已安装
go test ./internal/... -v
```

测试会自动创建临时 sandbox，在其中执行真实 git 命令并验证结果。不需要额外依赖。

## 架构

```
用户输入 git 命令
       ↓
   ┌─────────┐    ┌──────────┐    ┌──────────┐
   │ cmdexec │───→│ session  │───→│ verify   │
   │ 解析+验证│    │ 路径收容  │    │ 断言引擎  │
   └─────────┘    └──────────┘    └──────────┘
       ↑                               ↓
   ┌───────────────────────────────────────┐
   │           training.Service            │
   │  SelectNextTask / StartTask / Verify  │
   │         GetProgress / Store           │
   └───────────────────────────────────────┘
```

### 四个包

| 包 | 职责 |
|---|---|
| `internal/session` | Session 领域对象。路径收容基于 `filepath.EvalSymlinks` canonical path，沿最近存在祖先判定 |
| `internal/cmdexec` | 统一命令执行器。三层验证：路径收容 → transport 收容 → 子命令形态表白名单。Setup 和用户命令走同一路径 |
| `internal/verify` | GitState 断言引擎。18 种基础断言 + 8 种 DAG 图结构断言。纯函数 `(Assertion, SandboxRoot, CWD) → Result` |
| `internal/training` | 领域服务收口。选题/开始/验证/记录/进度 全在 Service 内，SQLite 持久化，HTTP/WS 只做薄适配 |

### Sandbox 隔离

每个任务在独立目录中运行，隔离宿主机 Git 配置：

- `GIT_CONFIG_GLOBAL` 指向 sandbox 内受控 `.gitconfig`
- `GIT_CONFIG_SYSTEM=/dev/null`
- 固定 `init.defaultBranch=main`、`user.name`、`user.email`
- 只继承 `PATH`，其余环境变量不泄漏

命令代数（非 denylist）保证隔离：

- **路径收容**: 所有路径参数 `EvalSymlinks` 后必须在 sandbox 内，symlink 逃逸被拒绝
- **Transport 收容**: `https://`、`ssh://`、`git://`、`file://`、`user@host:` 全部拒绝。L3 远程用 sandbox 内 bare repo 模拟
- **形态表白名单**: 每个 git 子命令定义允许的 flags，未列出的 flag 默认拒绝。`--template`、`remote set-url` 等危险形态显式封死

## 课纲

| Layer | 目标 | 核心内容 |
|-------|------|----------|
| L1 | 能保存工作 | init, add, commit, status, log, diff, .gitignore |
| L2 | 能在分支上协作 | branch, checkout/switch, merge(FF+3-way), 解冲突 |
| L3 | 能和远程交互 | clone, remote, push, pull, fetch |
| L4 | 能修复错误+并行工作 | amend, restore, reset, stash, worktree, reflog |
| L5 | 理解 agent 操作 | objects/refs/DAG, rebase, cherry-pick |

## 题库格式

题目定义在 `tasks/*.json`，每题包含 setup（初始化命令）和 verify（声明式断言）：

```json
{
  "L1.1": {
    "name": "初始化仓库",
    "tasks": [{
      "id": "L1.1-a",
      "description": "创建 Git 仓库，新建文件并完成第一次提交。",
      "setup": [],
      "verify": [
        { "type": "commit_count", "min": 1 },
        { "type": "file_exists", "path": "hello.txt" },
        { "type": "status_clean" }
      ],
      "hints": ["git init", "echo hello > hello.txt", "git add . && git commit -m 'first'"],
      "on_pass_note": "git init 创建 .git 目录，HEAD 指向 refs/heads/main。"
    }]
  }
}
```

Setup 命令走统一执行器（同样受命令代数约束），verify 断言是纯函数。

## 断言类型

**基础** (L1-L3):
`branch_exists` `branch_current` `commit_count` `commit_message_contains` `file_exists` `file_not_exists` `file_contains` `file_staged` `status_clean` `tag_exists` `remote_exists` `remote_url` `stash_count` `worktree_count` `config_equals` `ref_equals` `gitignore_ignores` `merge_clean` `file_count`

**DAG 图结构** (L4-L5，防止"文件对但历史错"的假通过):
`head_parents` `ref_ancestor_of` `ref_not_ancestor_of` `commit_tree_matches` `commit_parents` `commit_is_not` `linear_history` `ref_points_to`

## 测试

```bash
# 全部测试
go test ./internal/... -v

# 只跑竖切题（4 道端到端）
go test ./internal/training/ -run TestVerticalSlice -v

# 只跑逃逸测试（25 条）
go test ./internal/cmdexec/ -run TestEscape -v
```

## 目录结构

```
git-guider/
├── internal/
│   ├── session/session.go       # Session + 路径收容
│   ├── cmdexec/
│   │   ├── parse.go             # 命令行 tokenizer
│   │   ├── algebra.go           # 形态表 + 三层验证
│   │   ├── env.go               # 基线环境 + .gitconfig
│   │   ├── exec.go              # 统一执行器
│   │   └── exec_test.go         # 逃逸测试
│   ├── verify/verify.go         # 26 种断言
│   └── training/
│       ├── service.go           # 领域服务
│       ├── store.go             # SQLite
│       ├── taskbank.go          # 题库加载
│       ├── mastery.go           # Mastery 算法
│       └── service_test.go      # 竖切题测试
└── tasks/vertical.json          # 4 道竖切题
```

## 下一步

- [ ] Step 2: WebSocket + REST API (`internal/server/`)
- [ ] Step 3: Svelte 前端 (xterm.js + TaskPanel + LevelMap)
- [ ] Step 4: 题库扩充 (L1-L5 共 ~70 题)
- [ ] Step 5: 构建脚本 + CI
