import { motion, AnimatePresence } from 'framer-motion'
import { Send, Sparkles } from 'lucide-react'
import { useState, useCallback } from 'react'
import type { Message } from '../types'

const ChatArea = () => {
  const [messages, setMessages] = useState<Message[]>([
    {
      id: 1,
      role: 'assistant',
      content: '你好！我是CoDream AI助手。我已经完成了需求分析和架构设计，现在正在进行代码实现阶段。有什么我可以帮助你的吗？',
      timestamp: new Date(Date.now() - 300000)
    }
  ])
  const [input, setInput] = useState('')

  const handleSend = useCallback(() => {
    if (!input.trim()) return

    const newMessage: Message = {
      id: messages.length + 1,
      role: 'user',
      content: input,
      timestamp: new Date()
    }

    setMessages([...messages, newMessage])
    setInput('')

    setTimeout(() => {
      const aiResponse: Message = {
        id: messages.length + 2,
        role: 'assistant',
        content: '我正在处理你的请求...',
        timestamp: new Date()
      }
      setMessages(prev => [...prev, aiResponse])
    }, 1000)
  }, [input, messages.length])

  const handleInputChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setInput(e.target.value)
  }, [])

  const handleKeyPress = useCallback((e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') handleSend()
  }, [handleSend])

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      <div className="flex-1 overflow-y-auto p-6 space-y-4">
        <AnimatePresence>
          {messages.map((message, index) => (
            <motion.div
              key={message.id}
              initial={{ y: 20, opacity: 0 }}
              animate={{ y: 0, opacity: 1 }}
              exit={{ y: -20, opacity: 0 }}
              transition={{ delay: index * 0.1 }}
              className={`flex ${message.role === 'user' ? 'justify-end' : 'justify-start'}`}
            >
              <div className={`max-w-[70%] ${
                message.role === 'user'
                  ? 'bg-gradient-to-br from-deep-blue-600 to-ocean-blue-500 text-white'
                  : 'glass-effect border border-deep-blue-100/50'
              } rounded-2xl p-4 shadow-soft`}>
                {message.role === 'assistant' ? (
                  <div className="flex items-center gap-2 mb-2">
                    <div className="w-6 h-6 rounded-lg bg-gradient-to-br from-deep-blue-600 to-ocean-blue-500 flex items-center justify-center">
                      <Sparkles className="w-3 h-3 text-white" />
                    </div>
                    <span className="text-xs font-semibold text-deep-blue-700">
                      CoDream AI
                    </span>
                  </div>
                ) : null}
                <p className={`text-sm leading-relaxed ${
                  message.role === 'user' ? 'text-white' : 'text-deep-blue-900'
                }`}>
                  {message.content}
                </p>
                <span className={`text-xs mt-2 block ${
                  message.role === 'user' ? 'text-white/70' : 'text-deep-blue-400'
                }`}>
                  {message.timestamp.toLocaleTimeString('zh-CN', {
                    hour: '2-digit',
                    minute: '2-digit'
                  })}
                </span>
              </div>
            </motion.div>
          ))}
        </AnimatePresence>
      </div>

      {/* 输入区域 */}
      <motion.div
        initial={{ y: 20, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        transition={{ delay: 0.6 }}
        className="p-6 border-t border-deep-blue-100/50 glass-effect"
      >
        <div className="flex gap-3">
          <input
            type="text"
            value={input}
            onChange={handleInputChange}
            onKeyPress={handleKeyPress}
            placeholder="输入消息..."
            className="flex-1 px-4 py-3 rounded-xl border border-deep-blue-200 focus:border-deep-blue-500 focus:ring-2 focus:ring-deep-blue-500/20 outline-none transition-all bg-white text-deep-blue-900 placeholder-deep-blue-400"
          />
          <motion.button
            whileHover={{ scale: 1.05 }}
            whileTap={{ scale: 0.95 }}
            onClick={handleSend}
            className="px-6 py-3 rounded-xl bg-gradient-to-r from-deep-blue-600 to-ocean-blue-500 text-white font-medium shadow-soft hover:shadow-glow transition-all flex items-center gap-2"
          >
            <Send className="w-4 h-4" />
            发送
          </motion.button>
        </div>
      </motion.div>
    </div>
  )
}

export default ChatArea
