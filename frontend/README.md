# Frontend

前端基于 Vue 3、TypeScript、Vite、Pinia 与 Arco Design，负责桌面应用的管理后台界面。

## 开发命令

安装依赖：

```bash
npm install
```

本地开发：

```bash
npm run dev
```

类型检查并构建：

```bash
npm run build
```

## 目录约定

- `src/pages`: 页面级视图
- `src/components`: 通用组件
- `src/services`: API、桌面桥接与业务服务
- `src/stores`: Pinia 状态管理
- `src/types`: 类型定义
- `wailsjs`: Wails 自动生成的桥接代码

## 维护说明

- `src` 目录仅保留真实源码，不保留 TypeScript 编译生成的 `.js` / `.js.map`
- 业务代码优先维护 `.ts`、`.vue` 文件
- Wails 自动生成文件集中在 `wailsjs`，不要手动修改生成结果

## 图片上传流程

桌面端图片处理是"先上传、后引用"的流程，涉及三个主要阶段。理解此流程对避免"图片丢失"和"重复上传"问题至关重要。

### 阶段一：图片选择与本地预览

用户通过 `<MediaUploadField>` 或 `<RichContentEditor>` 选择图片后，**不上传到服务器**，而是通过 `FileReader.readAsDataURL()` 将文件转为 `data:image/...;base64,...` 格式保存在 `form.thumbnail` 或 `form.content` 中。

- `MediaUploadField.vue`：选中图片 → 读取为 data URL → `emit('update:modelValue', dataUrl)` → 表单字段更新
- `RichContentEditor.vue`：选择图片 → 读取为 data URL → `insertFn(dataUrl, ...)` → 编辑器内部显示图片

此时图片仅存在于当前页面的浏览器内存中，只有当前用户能看到。

### 阶段二：提交时上传（prepareSubmitPayload）

当用户点击"上传文章/同步文章"时，`handleSubmit()` → `prepareSubmitPayload()` 负责将所有 data URL 上传到服务器。

```
        prepareSubmitPayload()
          │
          ├── ① 缩略图处理
          │     ├── data:image → uploadDeferredDataUrl() 上传 → 替换为服务器 URL
          │     └── HTTP URL  → uploadRemoteMedia() 上传外部图片 → 替换为服务器 URL
          │
          ├── ② 获取编辑器最新 HTML
          │     └── contentEditorRef.getCurrentHtml()
          │
          └── ③ 正文 HTML 图片处理
                ├── uploadDeferredHtmlImages()  → 上传 data:image → 替换为服务器 URL
                └── （新建模式）uploadRemoteHtmlImages() → 上传外部 HTTP 图片
```

处理完成后，`form.content` 和 `form.thumbnail` 中的图片 URL 全部被替换为服务器 URL（如 `http://localhost/storage/images/1/editor/xxx.webp`）。

### 阶段三：同步后用服务端数据刷新

**编辑模式**下，`updateArticle()` 成功后需要**用服务端返回的最新数据刷新表单**，这是避免图片"丢失"的关键：

```
updateArticle(payload) → 成功返回 result
  └→ 用 result 填充 form（包含服务器 URL）
      └→ form.content 设为服务器返回的 HTML
          └→ wangEditor 调用 setHtml(服务器HTML)
              └→ 所有 `<img src="...">` 都是正确的服务器 URL
```

### 常见陷阱与规则

**规则 1：永远不要依赖本地拼接的 URL 作为最终值**

`prepareSubmitPayload` 中上传得到的 URL 只是"中间态"。同步后必须以服务端返回的 `result.content` / `result.thumbnail` 为准，因为这些字段经过了 `normaliseTenantHtmlAssets()` 的 URL 标准化处理。

**规则 2：同源 URL 不能当作"远程图片"重新上传**

如果图片 URL 的 `origin` 与 `apiBaseUrl` 的 `origin` 相同（如同属 `http://localhost`），说明图片已在本服务器上。`uploadRemoteMedia` 和 `uploadRemoteHtmlImages` **必须跳过**此类 URL，否则会导致重复上传产生重复文件。

对应代码位置：[`deferred-media.service.ts`] 中的 `isSameOriginAsApi()` 检查：

```typescript
function isSameOriginAsApi(url: string, apiBaseUrl: string): boolean {
  try {
    const urlOrigin = new URL(url).origin;
    const apiOrigin = new URL(apiBaseUrl).origin;
    return urlOrigin === apiOrigin;
  } catch {
    return false;
  }
}
```

**规则 3：`uploadDeferredHtmlImages` 只能处理 data URL**

该函数检查 `html.includes(DATA_URL_PREFIX)`，如果没有 data URL 则直接返回原 HTML。它不能处理 HTTP URL 的图片——那是 `uploadRemoteHtmlImages` 的职责。

**规则 4：`uploadRemoteHtmlImages` 仅在新建文章时执行**

编辑模式下不需要执行 `uploadRemoteHtmlImages`，因为已有文章中的图片已经是服务器 URL，而新上传的图片在 `uploadDeferredHtmlImages` 阶段已经处理完毕。
对应代码：`if (!isEditMode.value) { ... }`

**规则 5：编辑模式下新的 data URL 新图片必须在 `getCurrentHtml()` 之前替换**

`prepareSubmitPayload` 的正确顺序是：

```
获取最新编辑器 HTML       →  getCurrentHtml()
上传 data URL 图片        →  uploadDeferredHtmlImages()
上传外部远程图片(新建时)    →  uploadRemoteHtmlImages()
组装最终 payload          →  buildPayload()
```

错误顺序会导致编辑器 HTML 中的 data URL 未被替换就提交到服务器。

### 流程图

```
用户选择图片
     │
     ▼
FileReader.readAsDataURL(file)
     │
     ▼
data:image/png;base64,...   ← 仅浏览器内存中
     │
     ▼ (点击"上传文章")
prepareSubmitPayload()
     │
     ├── [缩略图] data URL? → uploadDeferredDataUrl() → POST /media/upload → 服务器 URL
     ├── [缩略图] HTTP URL? → uploadRemoteMedia()     → 同源? 跳过 | 非同源→下载→上传→URL
     │
     ├── getCurrentHtml() → 编辑器最新 HTML（含 data URL + 已有服务器 URL）
     │
     ├── uploadDeferredHtmlImages()
     │   └── 找到 img[src^="data:"] → POST /media/upload → 替换为服务器 URL
     │
     ├── （仅新建）uploadRemoteHtmlImages()
     │   └── 找到 img[src^="http"] → 同源? 跳过 | 非同源→下载→上传→替换 URL
     │
     └── buildPayload() → 组装为最终 payload
           │
           ▼ (编辑模式)
     updateArticle(payload)
           │
           ▼ (成功)
     用 result 刷新表单 form.* = result.*
     编辑器 setHtml(result.content) → 所有图片 URL 正确显示
```

### 关键文件职责

| 文件 | 职责 |
|------|------|
| `services/media.service.ts` | 底层上传：验证文件 → `UploadDesktopMedia`(Go 桥接) 或 FormData 上传 → 返回 `MediaItem` |
| `services/deferred-media.service.ts` | 中高层上传逻辑：data URL 上传、HTML 图片扫描替换、远程图片下载上传、同源跳过检测 |
| `components/MediaUploadField.vue` | 缩略图/单图组件：文件选择 → data URL → `v-model` 绑定到表单 |
| `components/RichContentEditor.vue` | 富文本编辑器：图片插入 → data URL → `v-model` 绑定到表单 |
| `pages/articles/ArticleEditorPage.vue` | 编排提交流程：`prepareSubmitPayload` + `handleSubmit` + 服务端刷新 |
