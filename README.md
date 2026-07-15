# EZC

`ezc`（easy copy）是一个适用于 Arch Linux 和 Windows 的终端文件剪切板，支持中英文混合路径。

## 使用

```text
ezc cp [-h|--hide] [文件或目录]  复制
ezc ct [-h|--hide] [文件或目录]  剪切
ezc pst [目标目录]               粘贴
ezc pad                          浏览剪切板
ezc rm [文件或目录]              移除索引
```

`cp`、`ct` 不传路径时会打开内嵌选择器；`pst` 不传目标时粘贴到当前目录。

## 安装

Arch Linux：

```bash
yay -S ezc-bin
```

Windows：解压 Release 中的 ZIP，将 `ezc.exe` 所在目录加入 `PATH`。

## 构建

```bash
make test
make package
```

项目采用 [MIT License](LICENSE)。详细设计见 [DESIGN.md](DESIGN.md)。
