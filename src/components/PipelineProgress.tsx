import { motion } from 'framer-motion'
import { CheckCircle2, Circle, Loader2, XCircle } from 'lucide-react'
import { useState } from 'react'
import type { PipelineStage } from '../types'

const PipelineProgress = () => {
  const [stages] = useState<PipelineStage[]>([
    { id: '1', name: '需求分析', status: 'completed', progress: 100 },
    { id: '2', name: '架构设计', status: 'completed', progress: 100 },
    { id: '3', name: '代码实现', status: 'running', progress: 68 },
    { id: '4', name: '测试验证', status: 'pending', progress: 0 },
    { id: '5', name: '部署上线', status: 'pending', progress: 0 },
  ])

  const getStatusIcon = (status: PipelineStage['status']) => {
    switch (status) {
      case 'completed':
        return <CheckCircle2 className="w-5 h-5 text-emerald-500" />
      case 'running':
        return <Loader2 className="w-5 h-5 text-ocean-blue-500 animate-spin" />
      case 'failed':
        return <XCircle className="w-5 h-5 text-red-500" />
      default:
        return <Circle className="w-5 h-5 text-deep-blue-300" />
    }
  }

  return (
    <motion.div
      initial={{ y: 20, opacity: 0 }}
      animate={{ y: 0, opacity: 1 }}
      transition={{ delay: 0.4, duration: 0.5 }}
      className="p-6 border-b border-deep-blue-100/50"
    >
      <h3 className="text-lg font-display font-semibold text-deep-blue-900 mb-4">
        Pipeline 进度
      </h3>
      <div className="flex items-center gap-3">
        {stages.map((stage, index) => (
          <motion.div
            key={stage.id}
            initial={{ scale: 0.8, opacity: 0 }}
            animate={{ scale: 1, opacity: 1 }}
            transition={{ delay: 0.5 + index * 0.1 }}
            className="flex-1"
          >
            <div className="flex items-center gap-2 mb-2">
              {getStatusIcon(stage.status)}
              <span className={`text-sm font-medium ${
                stage.status === 'completed' ? 'text-emerald-600' :
                stage.status === 'running' ? 'text-ocean-blue-600' :
                'text-deep-blue-400'
              }`}>
                {stage.name}
              </span>
            </div>
            <div className="h-2 bg-deep-blue-100 rounded-full overflow-hidden">
              <motion.div
                initial={{ width: 0 }}
                animate={{ width: `${stage.progress}%` }}
                transition={{ delay: 0.6 + index * 0.1, duration: 0.8 }}
                className={`h-full ${
                  stage.status === 'completed' ? 'bg-emerald-500' :
                  stage.status === 'running' ? 'bg-gradient-to-r from-ocean-blue-500 to-deep-blue-600' :
                  'bg-deep-blue-300'
                }`}
              />
            </div>
          </motion.div>
        ))}
      </div>
    </motion.div>
  )
}

export default PipelineProgress
