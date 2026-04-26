import { motion } from 'framer-motion'
import {
  Sparkles,
  FolderKanban,
  History,
  Settings,
  Plus,
  ChevronRight
} from 'lucide-react'
import { useState, useCallback } from 'react'
import type { Project } from '../types'

const Sidebar = () => {
  const [activeItem, setActiveItem] = useState('projects')
  const [projects] = useState<Project[]>([
    { id: 1, name: 'AI助手优化', status: 'active', progress: 65 },
    { id: 2, name: '数据分析平台', status: 'active', progress: 42 },
    { id: 3, name: '移动端重构', status: 'paused', progress: 28 },
  ])

  const handleMenuClick = useCallback((itemId: string) => {
    setActiveItem(itemId)
  }, [])

  const menuItems = [
    { id: 'projects', icon: FolderKanban, label: '项目列表' },
    { id: 'history', icon: History, label: '历史记录' },
    { id: 'settings', icon: Settings, label: '设置' },
  ]

  return (
    <motion.aside
      initial={{ x: -300, opacity: 0 }}
      animate={{ x: 0, opacity: 1 }}
      transition={{ duration: 0.5, ease: [0.22, 1, 0.36, 1] }}
      className="w-80 h-full glass-effect border-r border-deep-blue-100/50 flex flex-col"
    >
      {/* Logo区域 */}
      <div className="p-6 border-b border-deep-blue-100/50">
        <motion.div
          initial={{ scale: 0.9, opacity: 0 }}
          animate={{ scale: 1, opacity: 1 }}
          transition={{ delay: 0.2, duration: 0.4 }}
          className="flex items-center gap-3"
        >
          <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-deep-blue-600 to-ocean-blue-500 flex items-center justify-center shadow-glow">
            <Sparkles className="w-5 h-5 text-white" />
          </div>
          <div>
            <h1 className="text-xl font-display font-bold gradient-text">
              CoDream
            </h1>
            <p className="text-xs text-deep-blue-400 font-medium">
              项目研发引擎
            </p>
          </div>
        </motion.div>
      </div>

      {/* 导航菜单 */}
      <nav className="flex-1 p-4 overflow-y-auto">
        <div className="space-y-1 mb-6">
          {menuItems.map((item, index) => {
            const Icon = item.icon
            const isActive = activeItem === item.id
            return (
              <motion.button
                key={item.id}
                initial={{ x: -20, opacity: 0 }}
                animate={{ x: 0, opacity: 1 }}
                transition={{ delay: 0.1 * index + 0.3 }}
                onClick={() => handleMenuClick(item.id)}
                className={`w-full flex items-center gap-3 px-4 py-3 rounded-xl transition-all duration-200 ${
                  isActive
                    ? 'bg-deep-blue-600 text-white shadow-soft'
                    : 'text-deep-blue-600 hover:bg-deep-blue-50'
                }`}
              >
                <Icon className="w-5 h-5" />
                <span className="font-medium">{item.label}</span>
              </motion.button>
            )
          })}
        </div>

        {/* 项目列表 */}
        <div className="space-y-3">
          <div className="flex items-center justify-between px-2">
            <h3 className="text-sm font-semibold text-deep-blue-700">
              活跃项目
            </h3>
            <button className="p-1 hover:bg-deep-blue-50 rounded-lg transition-colors">
              <Plus className="w-4 h-4 text-deep-blue-600" />
            </button>
          </div>
          {projects.map((project, index) => (
            <motion.div
              key={project.id}
              initial={{ y: 20, opacity: 0 }}
              animate={{ y: 0, opacity: 1 }}
              transition={{ delay: 0.1 * index + 0.5 }}
              className="p-3 rounded-xl bg-white border border-deep-blue-100/50 hover:shadow-soft transition-all duration-200 cursor-pointer group"
            >
              <div className="flex items-start justify-between mb-2">
                <h4 className="text-sm font-semibold text-deep-blue-900 group-hover:text-deep-blue-700">
                  {project.name}
                </h4>
                <ChevronRight className="w-4 h-4 text-deep-blue-400 opacity-0 group-hover:opacity-100 transition-opacity" />
              </div>
              <div className="flex items-center gap-2">
                <div className="flex-1 h-1.5 bg-deep-blue-100 rounded-full overflow-hidden">
                  <motion.div
                    initial={{ width: 0 }}
                    animate={{ width: `${project.progress}%` }}
                    transition={{ delay: 0.1 * index + 0.7, duration: 0.8 }}
                    className="h-full bg-gradient-to-r from-deep-blue-600 to-ocean-blue-500"
                  />
                </div>
                <span className="text-xs font-medium text-deep-blue-600">
                  {project.progress}%
                </span>
              </div>
            </motion.div>
          ))}
        </div>
      </nav>
    </motion.aside>
  )
}

export default Sidebar
