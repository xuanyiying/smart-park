import { useState } from 'react'
import IndoorMap from './components/IndoorMap'
import ParkingStatus from './components/ParkingStatus'
import AnalyticsDashboard from './components/AnalyticsDashboard'
import 'leaflet/dist/leaflet.css'
import './App.css'
import './components/AnalyticsDashboard.css'

function App() {
  const [activeTab, setActiveTab] = useState('navigation')

  return (
    <div className="app">
      <header className="header">
        <h1>智慧停车系统</h1>
        <nav className="nav">
          <button 
            className={`nav-button ${activeTab === 'navigation' ? 'active' : ''}`}
            onClick={() => setActiveTab('navigation')}
          >
            车位导航
          </button>
          <button 
            className={`nav-button ${activeTab === 'analytics' ? 'active' : ''}`}
            onClick={() => setActiveTab('analytics')}
          >
            数据分析
          </button>
        </nav>
      </header>
      
      {activeTab === 'navigation' ? (
        <>
          <ParkingStatus />
          <main className="main">
            <IndoorMap />
          </main>
        </>
      ) : (
        <main className="main">
          <AnalyticsDashboard />
        </main>
      )}
    </div>
  )
}

export default App
