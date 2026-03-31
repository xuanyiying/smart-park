# Smart Park Logo 设计规范

## 设计理念

### 核心概念
- **P**：停车（Parking）的核心符号，全球通用识别
- **信号波**：代表智能（Smart）、连接、科技感
- **蓝色渐变**：专业、可信、科技感

### 设计元素

1. **主图标**
   - 圆形背景，蓝色渐变
   - 中心白色大写字母 "P"
   - 顶部三个信号点，象征智能连接
   - 弧线装饰，增加动感

2. **配色方案**
   - 主色：`#3B82F6` (蓝色 500)
   - 深色：`#1E40AF` (蓝色 800)
   - 浅色：`#60A5FA` (蓝色 400)
   - 白色：`#FFFFFF` (用于文字和点缀)

3. **字体**
   - 英文：Arial Bold / Inter Bold
   - 中文：系统默认无衬线字体

## 文件清单

| 文件 | 用途 | 尺寸 |
|------|------|------|
| `logo.svg` | 主 Logo，圆形带背景 | 200x200 |
| `logo-icon.svg` | 图标版本，方形圆角 | 64x64 |
| `logo-horizontal.svg` | 横版 Logo + 文字 | 400x100 |

## 使用场景

### GitHub README
```markdown
<p align="center">
  <img src="docs/assets/logo.svg" alt="Smart Park Logo" width="200">
</p>
```

### Favicon / 小图标
使用 `logo-icon.svg`，尺寸 32x32 或 64x64

### 文档头部
使用 `logo-horizontal.svg`，适合放在文档顶部

## 导出 PNG（如需）

使用以下工具导出不同尺寸的 PNG：

```bash
# 使用 ImageMagick
convert -background none logo.svg -resize 200x200 logo.png
convert -background none logo.svg -resize 512x512 logo-512.png

# 或使用在线工具
# https://convertio.co/svg-png/
```

## 变体设计（可选）

### 深色模式版本
- 背景：深蓝到黑色渐变
- 文字：保持白色

### 单色版本
- 仅使用 `#3B82F6` 单一颜色
- 适用于打印或特殊场景

### 反白版本
- 深色背景上使用白色 Logo
- 适用于深色主题或特殊背景
