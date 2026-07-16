# EZC

`ezc`（easy copy）是一个适用于 Arch Linux 和 Windows 的终端文件剪切板，支持中英文混合路径。

## 使用

```text
ezc cp [-h|--hide] [文件、目录或查询]  复制
ezc ct [-h|--hide] [文件、目录或查询]  剪切
ezc pst [目标目录]               粘贴
ezc pad                          浏览剪切板
ezc rm [文件或目录]              移除索引
```

`cp`、`ct` 传入的路径存在时会直接操作；路径不存在时会把参数作为当前目录查询，按“完全匹配、子串匹配、子序列匹配”的顺序自动选择最佳结果，同级匹配保留当前目录列表顺序。例如 `ezc cp csbg` 可以直接匹配并复制 `测试报告.txt`，无需打开菜单。

不传参数时仍会打开内嵌选择器；在选择器中可直接输入英文、中文名称的全拼或拼音首字母进行相同规则的模糊搜索，按退格键修改搜索词，按 `Esc` 退出。`-h`/`--hide` 会在菜单和参数搜索中包含隐藏项目。`pst` 不传目标时粘贴到当前目录。

## 安装

Arch Linux：

```bash
yay -S ezc-bin
```

Windows（Scoop）：

```powershell
scoop bucket add ezc https://github.com/mornhussakuyo-hub/scoop-ezc
scoop install ezc/ezc
```

## 构建

```bash
make test
make package
```

项目采用 [MIT License](LICENSE)。详细设计见 [DESIGN.md](DESIGN.md)。
