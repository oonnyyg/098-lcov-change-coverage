# covergate

合并多个 CI 分片产生的 LCOV 行覆盖数据，结合变更行清单判断本次变更是否达到覆盖门槛。

## 用法

```
go run ./cmd/covergate --config 任务.json
```

退出码：`0` 门禁通过；`1` 门槛未达或存在缺失数据（报告已写出）；`2` 输入非法（不写报告）。

## 任务配置（JSON）

```json
{
  "lcov_files": ["shard1.info", "shard2.info"],
  "changes_file": "changes.json",
  "overall_threshold": 8000,
  "change_threshold": 9000,
  "output": "report.json"
}
```

- `lcov_files`：LCOV 文件列表，相对路径相对于配置文件所在目录解析。
- `changes_file`：变更行清单路径。
- `overall_threshold` / `change_threshold`：总体与变更门槛，0–10000 的整数万分比（如 `8000` 表示 80%）。按精确比例比较，相等视为通过。
- `output`：报告输出路径。

## 变更行清单（JSON）

```json
{
  "src/a.go": [10, 11, 25],
  "src/b.go": [3]
}
```

键为正斜杠相对源码路径（无空段、无 `.`/`..` 段，大小写敏感），值为新版文件中的正整数行号，重复行自动去重。

## LCOV 子集

仅支持 `SF:`、`DA:<行号>,<次数>` 与 `end_of_record`；`TN:`、`LF:`、`LH:` 被忽略，其他记录报错。`DA` 不允许校验和字段，行号为正整数、次数为非负整数。段未闭合、段嵌套、段外 `DA`、非法计数均报错并定位行号。同一路径同一行的次数跨段、跨文件相加；出现过的行计为可执行行，合计次数大于零才算覆盖。

## 报告格式（JSON）

```json
{
  "passed": true,
  "overall": {
    "path": "",
    "covered": 8, "total": 10, "ratio": "8/10",
    "change_covered": 3, "change_total": 4, "change_ratio": "3/4",
    "uncovered_change_lines": [],
    "non_executable_lines": []
  },
  "files": [
    {
      "path": "src/a.go",
      "covered": 8, "total": 10, "ratio": "8/10",
      "change_covered": 3, "change_total": 4, "change_ratio": "3/4",
      "uncovered_change_lines": [25],
      "non_executable_lines": [40]
    }
  ],
  "missing_files": []
}
```

- `ratio` / `change_ratio`：精确分数 `"覆盖数/总数"`；分母为零时为 `null`，且该门槛视为通过。
- `uncovered_change_lines`：变更行中可执行但未覆盖的行，升序。
- `non_executable_lines`：变更行中不属于可执行行的行，单独列出且不计入分母。
- `missing_files`：变更文件中完全没有任何覆盖段的文件；非空时门禁失败。
- `files` 按路径字典序稳定排序。

## 测试

```
go test ./...
```
