export interface Project {
  id: number
  name: string
  status: 'active' | 'paused' | 'completed'
  progress: number
}

export interface PipelineStage {
  id: string
  name: string
  status: 'pending' | 'running' | 'completed' | 'failed'
  progress: number
}

export interface Message {
  id: number
  role: 'user' | 'assistant'
  content: string
  timestamp: Date
}
