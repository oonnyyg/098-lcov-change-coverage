# covergate 格式说明

`covergate` 合并多个 CI 分片产生的 LCOV 行覆盖数据，结合变更行清单判定覆盖门禁。

运行方式：

```
go run ./cmd/covergate --config 任务.json
```

退出码：`0` 门禁通过；`1` 门禁失败（报告仍写出）；`2` 输入非法（不写报告）。

## 任务配置（--config 指向的 JSON）

```json
{
  "lcov_files": ["shard1.info", "shard2.info"],
  "changes_file": "changes.json",
  "output_file": "report.json",
  "overall_threshold": 8000,
  "change_threshold": 9000
}
```

- `lcov_files`：LCOV 文件列表，至少一个。
- `changes_file`：变更行清单 JSON 路径。
- `output_file`：报告输出路径。
- `overall_threshold` / `change_threshold`：总体 / 变更覆盖门槛，0–10000 的整数万分比。
- 所有相对路径均相对于配置文件所在目录解析。

## 变更行清单

顶层为对象，键为源文件路径，值为该文件新版中变更行的正整数行号数组（重复行自动去重）：

```json
{
  "src/a.go": [10, 11, 25],
  "src/b.go": [3]
}
```

## LCOV 输入约束

- 仅支持 `SF:<path>`、`DA:<line>,<count>`、`end_of_record` 三种记录；`TN:`、`LF:`、`LH:` 被忽略，其余记录报错。
- `SF` 开启段，`end_of_record` 关闭段；段未闭合、段嵌套、段外 `DA` 均报错并给出文件与行号。
- `DA` 行号为正整数、次数为非负整数，不允许校验和等额外字段。
- 路径为正斜杠相对路径，不含空段或点段（`.`/`..`），大小写敏感、按原样保留。
- 同路径同一行的次数跨段、跨文件相加；出现过的行计为可执行行，合计次数大于零才算覆盖。

## 门禁规则

- 变更行与可执行行的交集参与变更覆盖率；其余变更行（非可执行行）单独列出且不计入分母。
- 变更文件完全没有覆盖段时记为缺失数据（`missing_files`），门禁失败；该规则在分母为零时仍然生效。
- 门槛按精确比例比较（整数运算 `covered*10000 >= threshold*total`），相等视为通过。
- 分母为零时比例为 `null` 且对应门槛视为通过。

## 报告格式（output_file）

```json
{
  "pass": true,
  "overall": {"covered": 3, "total": 4, "ratio": 0.75, "threshold": 8000, "pass": false},
  "change":  {"covered": 2, "total": 3, "ratio": 0.666..., "threshold": 9000, "pass": false},
  "files": [
    {
      "path": "src/a.go",
      "covered": 2, "total": 3, "ratio": 0.666...,
      "changed_covered": 1, "changed_total": 2, "changed_ratio": 0.5,
      "uncovered_changed_lines": [2],
      "non_executable_changed_lines": [99]
    }
  ],
  "missing_files": ["src/c.go"]
}
```

- `files` 按路径字典序稳定排序，行号数组升序。
- `ratio` / `changed_ratio` 在分母为零时为 `null`。
- 顶层 `pass` 要求总体门槛、变更门槛均通过且无缺失文件。
