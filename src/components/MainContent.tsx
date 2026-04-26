import { motion } from 'framer-motion'
import PipelineProgress from './PipelineProgress'
import ChatArea from './ChatArea'

const MainContent = () => {
  return (
    <main className="flex-1 flex flex-col h-full overflow-hidden">
      <motion.div
        initial={{ y: -20, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        transition={{ delay: 0.3, duration: 0.5 }}
        className="p-6 border-b border-deep-blue-100/50 glass-effect"
      >
        <h2 className="text-2xl font-display font-bold text-deep-blue-900">
          AI助手优化
        </h2>
        <p className="text-sm text-deep-blue-500 mt-1">
          正在进行中 · 最后更新于 2分钟前
        </p>
      </motion.div>

      <div className="flex-1 flex flex-col overflow-hidden">
        <PipelineProgress />
        <ChatArea />
      </div>
    </main>
  )
}

export default MainContent
