# CoDream Dashboard

一个精致的React项目管理Dashboard应用，采用"精致现代主义"设计风格。

## 技术栈

- **React 18** - 现代化的UI框架
- **TypeScript** - 类型安全
- **Vite** - 快速的构建工具
- **Tailwind CSS** - 实用优先的CSS框架
- **Framer Motion** - 流畅的动画库
- **Lucide React** - 精美的图标库

## 设计特点

### 美学方向：精致现代主义（Refined Modernism）

- **字体系统**
  - 标题：Satoshi（独特的几何无衬线字体）
  - 正文：Geist（现代化的可读性字体）

- **色彩系统**
  - 主色：深邃蓝色调（Deep Blue #1E40AF）
  - 强调色：海洋蓝（Ocean Blue #0EA5E9）
  - 背景：极浅灰蓝（#F8FAFC）

- **视觉效果**
  - 玻璃态效果（Glass morphism）
  - 渐变网格背景
  - 柔和阴影和发光效果
  - 流畅的页面加载动画

## 快速开始

### 安装依赖

\`\`\`bash
npm install
\`\`\`

### 启动开发服务器

\`\`\`bash
npm run dev
\`\`\`

应用将在 http://localhost:3000 自动打开。

### 构建生产版本

\`\`\`bash
npm run build
\`\`\`

### 预览生产构建

\`\`\`bash
npm run preview
\`\`\`

## 项目结构

\`\`\`
CoDream/
├── src/
│   ├── components/
│   │   ├── Sidebar.tsx          # 左侧导航栏
│   │   ├── MainContent.tsx      # 主内容区
│   │   ├── PipelineProgress.tsx # Pipeline进度展示
│   │   └── ChatArea.tsx         # 对话区域
│   ├── types/
│   │   └── index.ts             # TypeScript类型定义
│   ├── App.tsx                  # 根组件
│   ├── main.tsx                 # 应用入口
│   └── index.css                # 全局样式
├── index.html
├── vite.config.ts
├── tailwind.config.js
├── tsconfig.json
└── package.json
\`\`\`

## 功能特性

- ✨ 精致的UI设计，避免通用AI美学
- 🎨 独特的字体和色彩系统
- 🌊 流畅的动画和微交互
- 📊 实时Pipeline进度展示
- 💬 智能对话界面
- 📱 响应式设计
- ⚡ 快速的开发体验

## 开发说明

项目使用TypeScript进行类型安全开发，所有组件都有完整的类型定义。使用Framer Motion实现流畅的动画效果，Tailwind CSS提供快速的样式开发体验。
