import { defineConfig } from 'vitepress'
import type { DefaultTheme } from 'vitepress'

const sharedHead = [['link', { rel: 'icon', href: '/favicon.svg' }]]

const sharedSocial = [
  { icon: 'github', link: 'https://github.com/colinleefish/rmb' },
] satisfies DefaultTheme.SocialLink[]

/** One sidebar for the whole site — VitePress reference style (grouped, always expanded). */
function unifiedEnSidebar(): DefaultTheme.SidebarItem[] {
  return [
    {
      text: 'Introduction',
      collapsed: false,
      items: [
        { text: 'What is rmb?', link: '/concept/' },
        { text: 'Getting started', link: '/guide/getting-started' },
      ],
    },
    {
      text: 'Concept',
      collapsed: false,
      items: [
        { text: 'URI scheme', link: '/concept/uri-scheme' },
        { text: 'The pyramid (T0–T3)', link: '/concept/pyramid' },
        { text: 'Sessions', link: '/concept/sessions' },
        { text: 'Turns', link: '/concept/turns' },
        { text: 'Atoms', link: '/concept/atoms' },
        { text: 'Scenes', link: '/concept/scenes' },
        { text: 'Long-term memories', link: '/concept/memories' },
        { text: 'How data flows', link: '/concept/pipeline' },
      ],
    },
    {
      text: 'Guide',
      collapsed: false,
      items: [
        { text: 'CLI for agents', link: '/guide/cli-for-agents' },
        { text: 'Corrections', link: '/guide/corrections' },
      ],
    },
    {
      text: 'Design',
      collapsed: false,
      items: [
        { text: 'Overview', link: '/design/' },
        { text: 'L0→L3 distillation', link: '/design/l0-l3' },
      ],
    },
    {
      text: 'Reference',
      collapsed: false,
      items: [
        { text: 'Entity model', link: '/reference/entity-model' },
        { text: 'Implementation plan', link: '/reference/plan' },
        { text: 'Deploy', link: '/reference/deploy' },
      ],
    },
  ]
}

function unifiedZhSidebar(): DefaultTheme.SidebarItem[] {
  return [
    {
      text: '入门',
      collapsed: false,
      items: [
        { text: '什么是 rmb？', link: '/zh/concept/' },
        { text: '快速开始', link: '/zh/guide/getting-started' },
      ],
    },
    {
      text: '理念',
      collapsed: false,
      items: [
        { text: 'URI 方案', link: '/zh/concept/uri-scheme' },
        { text: '金字塔（T0–T3）', link: '/zh/concept/pyramid' },
        { text: '会话（Session）', link: '/zh/concept/sessions' },
        { text: '轮次（Turn）', link: '/zh/concept/turns' },
        { text: '原子（Atom）', link: '/zh/concept/atoms' },
        { text: '场景（Scene）', link: '/zh/concept/scenes' },
        { text: '长期记忆（Memory）', link: '/zh/concept/memories' },
        { text: '数据如何流动（Pipeline）', link: '/zh/concept/pipeline' },
      ],
    },
    {
      text: '指南',
      collapsed: false,
      items: [
        { text: 'Agent 用 CLI', link: '/zh/guide/cli-for-agents' },
        { text: '人工纠正', link: '/zh/guide/corrections' },
      ],
    },
    {
      text: '设计',
      collapsed: false,
      items: [
        { text: '概览', link: '/zh/design/' },
        { text: 'L0→L3 知识蒸馏', link: '/zh/design/l0-l3' },
        { text: '整合策略评述', link: '/zh/design/consolidation' },
      ],
    },
    {
      text: '参考',
      collapsed: false,
      items: [
        { text: '实体模型', link: '/zh/reference/entity-model' },
        { text: '实施计划', link: '/zh/reference/plan' },
        { text: '部署', link: '/zh/reference/deploy' },
      ],
    },
  ]
}

export default defineConfig({
  title: 'rmb',
  head: sharedHead,
  locales: {
    root: {
      label: 'English',
      lang: 'en-US',
      description: 'Long-term memory for AI agent conversations',
      themeConfig: {
        logo: '/favicon.svg',
        nav: [
          { text: 'Concept', link: '/concept/', activeMatch: '/concept/' },
          { text: 'Guide', link: '/guide/getting-started', activeMatch: '/guide/' },
          { text: 'Design', link: '/design/', activeMatch: '/design/' },
          { text: 'Reference', link: '/reference/entity-model', activeMatch: '/reference/' },
        ],
        sidebar: { '/': unifiedEnSidebar() },
        socialLinks: sharedSocial,
        footer: {
          message: 'Personal long-term memory for AI agents',
          copyright: 'Copyright © Colin Lee Fish',
        },
        search: { provider: 'local' },
        docFooter: { prev: 'Previous', next: 'Next' },
        outline: { label: 'On this page', level: [2, 3] },
        sidebarMenuLabel: 'Menu',
        returnToTopLabel: 'Back to top',
        darkModeSwitchLabel: 'Appearance',
        langMenuLabel: 'Change language',
      },
    },
    zh: {
      label: '简体中文',
      lang: 'zh-CN',
      link: '/zh/',
      description: 'AI 智能体对话的长期记忆',
      themeConfig: {
        logo: '/favicon.svg',
        nav: [
          { text: '理念', link: '/zh/concept/', activeMatch: '/zh/concept/' },
          { text: '指南', link: '/zh/guide/getting-started', activeMatch: '/zh/guide/' },
          { text: '设计', link: '/zh/design/', activeMatch: '/zh/design/' },
          { text: '参考', link: '/zh/reference/entity-model', activeMatch: '/zh/reference/' },
        ],
        sidebar: { '/zh/': unifiedZhSidebar() },
        socialLinks: sharedSocial,
        footer: {
          message: '面向 AI 智能体的个人长期记忆',
          copyright: 'Copyright © Colin Lee Fish',
        },
        search: { provider: 'local' },
        docFooter: { prev: '上一页', next: '下一页' },
        outline: { label: '本页目录', level: [2, 3] },
        returnToTopLabel: '返回顶部',
        sidebarMenuLabel: '菜单',
        darkModeSwitchLabel: '主题',
        langMenuLabel: '语言',
      },
    },
  },
})
