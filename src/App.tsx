import { motion } from 'framer-motion'
import Sidebar from './components/Sidebar'
import MainContent from './components/MainContent'

function App() {
  return (
    <div className="flex h-screen overflow-hidden bg-gradient-mesh">
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 0.6 }}
        className="flex w-full"
      >
        <Sidebar />
        <MainContent />
      </motion.div>
    </div>
  )
}

export default App
