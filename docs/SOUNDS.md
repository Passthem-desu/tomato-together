# 提示音添加指南

## 文件位置

所有提示音逻辑在 `frontend/src/lib/sounds.ts`。

## 如何添加新提示音

### 1. 编写音效生成函数

在文件的 "Built-in sound generators" 区域添加一个新函数：

```typescript
/** 我的新音效 — 简短描述 */
function genMySound(ctx: AudioContext) {
  // 使用 Web Audio API 构建音效
  beep(ctx, 440, 0.3, 0, 0.1);  // 频率 440Hz，持续 0.3s，延迟 0s，音量 0.1
  beep(ctx, 660, 0.2, 0.4, 0.1); // 第二个音在 0.4s 后
}
```

### 2. 注册到音效表

在 `BUILTIN_SOUNDS` 对象中添加条目：

```typescript
const BUILTIN_SOUNDS: Record<string, SoundDef> = {
  // ... 已有条目 ...
  my_sound: { name: 'my_sound', label: '我的音效', play: genMySound },
};
```

| 字段 | 说明 |
|------|------|
| `name` | 内部 key（英文，snake_case，唯一） |
| `label` | 显示名称（会出现在设置 UI 中） |
| `play` | 音效生成函数 |

### 3. Web Audio 辅助函数

`sounds.ts` 内置了 `beep()` 辅助函数：

```typescript
function beep(
  ctx: AudioContext,
  freq: number,       // 频率（Hz），如 440、660、880
  duration: number,   // 持续秒数
  startDelay = 0,     // 起始延迟秒数
  vol = 0.12          // 音量（0~1），建议 0.05~0.2
)
```

可以使用多个 `beep()` 组合成和弦/旋律：

```typescript
function genChord(ctx: AudioContext) {
  beep(ctx, 523, 0.5, 0);   // C
  beep(ctx, 659, 0.5, 0);   // E
  beep(ctx, 784, 0.5, 0);   // G  — 三音同时响
}
```

### 4. 默认事件映射

在 `DEFAULT_PREFS` 中可设置新音效的默认事件：

```typescript
const DEFAULT_PREFS: SoundPrefs = {
  focus_start: 'chime_up',     // 专注开始
  focus_end:   'chime_down',   // 专注结束
  rest_end:    'double_beep',  // 休息结束
  all_done:    'chime_up',     // 全部完成
};
```

音效 key 对应 `BUILTIN_SOUNDS` 中的 `name` 字段。

### 5. 可用事件

| 事件 key | 触发时机 |
|----------|---------|
| `focus_start` | 点击"开始番茄" |
| `focus_end` | 专注时间结束（自动或手动） |
| `rest_end` | 休息时间结束 |
| `all_done` | 全部番茄完成 |

### 6. 用户偏好存储

用户选择的音效存储在 `localStorage` 的 `tomatogether_sounds` 键中，格式为 `{ focus_start: "chime_up", ... }`。
