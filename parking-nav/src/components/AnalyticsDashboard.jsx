import React, { useState, useEffect } from 'react';
import Chart from 'chart.js/auto';
import './AnalyticsDashboard.css';

const AnalyticsDashboard = () => {
  const [revenueData, setRevenueData] = useState([]);
  const [occupancyData, setOccupancyData] = useState([]);
  const [vehicleFlowData, setVehicleFlowData] = useState([]);
  const [peakHoursData, setPeakHoursData] = useState([]);
  const [loading, setLoading] = useState(true);

  // Mock data for demonstration
  const mockRevenueData = [
    { date: '2024-01-01', revenue: 1200, vehicleCount: 150 },
    { date: '2024-01-02', revenue: 1350, vehicleCount: 170 },
    { date: '2024-01-03', revenue: 1100, vehicleCount: 140 },
    { date: '2024-01-04', revenue: 1450, vehicleCount: 180 },
    { date: '2024-01-05', revenue: 1600, vehicleCount: 200 },
    { date: '2024-01-06', revenue: 1800, vehicleCount: 220 },
    { date: '2024-01-07', revenue: 1300, vehicleCount: 160 },
  ];

  const mockOccupancyData = [
    { timestamp: '2024-01-07T00:00:00Z', rate: 0.1, occupiedSpaces: 10, totalSpaces: 100 },
    { timestamp: '2024-01-07T04:00:00Z', rate: 0.05, occupiedSpaces: 5, totalSpaces: 100 },
    { timestamp: '2024-01-07T08:00:00Z', rate: 0.7, occupiedSpaces: 70, totalSpaces: 100 },
    { timestamp: '2024-01-07T12:00:00Z', rate: 0.8, occupiedSpaces: 80, totalSpaces: 100 },
    { timestamp: '2024-01-07T16:00:00Z', rate: 0.7, occupiedSpaces: 70, totalSpaces: 100 },
    { timestamp: '2024-01-07T20:00:00Z', rate: 0.9, occupiedSpaces: 90, totalSpaces: 100 },
    { timestamp: '2024-01-07T23:59:59Z', rate: 0.2, occupiedSpaces: 20, totalSpaces: 100 },
  ];

  const mockVehicleFlowData = [
    { timestamp: '2024-01-07T00:00:00Z', entries: 5, exits: 3, netFlow: 2 },
    { timestamp: '2024-01-07T04:00:00Z', entries: 2, exits: 1, netFlow: 1 },
    { timestamp: '2024-01-07T08:00:00Z', entries: 30, exits: 10, netFlow: 20 },
    { timestamp: '2024-01-07T12:00:00Z', entries: 25, exits: 20, netFlow: 5 },
    { timestamp: '2024-01-07T16:00:00Z', entries: 35, exits: 25, netFlow: 10 },
    { timestamp: '2024-01-07T20:00:00Z', entries: 40, exits: 15, netFlow: 25 },
    { timestamp: '2024-01-07T23:59:59Z', entries: 8, exits: 5, netFlow: 3 },
  ];

  const mockPeakHoursData = [
    { startHour: 8, endHour: 10, expectedVehicles: 80, probability: 0.9 },
    { startHour: 12, endHour: 13, expectedVehicles: 85, probability: 0.85 },
    { startHour: 17, endHour: 19, expectedVehicles: 90, probability: 0.95 },
  ];

  useEffect(() => {
    // Simulate API calls
    setTimeout(() => {
      setRevenueData(mockRevenueData);
      setOccupancyData(mockOccupancyData);
      setVehicleFlowData(mockVehicleFlowData);
      setPeakHoursData(mockPeakHoursData);
      setLoading(false);
    }, 1000);
  }, []);

  useEffect(() => {
    if (!loading) {
      // Initialize revenue chart
      const revenueCtx = document.getElementById('revenueChart');
      if (revenueCtx) {
        new Chart(revenueCtx, {
          type: 'bar',
          data: {
            labels: revenueData.map(item => item.date),
            datasets: [
              {
                label: 'Revenue (¥)',
                data: revenueData.map(item => item.revenue),
                backgroundColor: 'rgba(54, 162, 235, 0.6)',
                borderColor: 'rgba(54, 162, 235, 1)',
                borderWidth: 1
              },
              {
                label: 'Vehicle Count',
                data: revenueData.map(item => item.vehicleCount),
                backgroundColor: 'rgba(75, 192, 192, 0.6)',
                borderColor: 'rgba(75, 192, 192, 1)',
                borderWidth: 1,
                yAxisID: 'y1'
              }
            ]
          },
          options: {
            responsive: true,
            scales: {
              y: {
                beginAtZero: true,
                title: {
                  display: true,
                  text: 'Revenue (¥)'
                }
              },
              y1: {
                beginAtZero: true,
                position: 'right',
                title: {
                  display: true,
                  text: 'Vehicle Count'
                },
                grid: {
                  drawOnChartArea: false
                }
              }
            }
          }
        });
      }

      // Initialize occupancy chart
      const occupancyCtx = document.getElementById('occupancyChart');
      if (occupancyCtx) {
        new Chart(occupancyCtx, {
          type: 'line',
          data: {
            labels: occupancyData.map(item => new Date(item.timestamp).toLocaleTimeString()),
            datasets: [
              {
                label: 'Occupancy Rate (%)',
                data: occupancyData.map(item => item.rate * 100),
                borderColor: 'rgba(255, 99, 132, 1)',
                backgroundColor: 'rgba(255, 99, 132, 0.1)',
                tension: 0.4,
                fill: true
              }
            ]
          },
          options: {
            responsive: true,
            scales: {
              y: {
                beginAtZero: true,
                max: 100,
                title: {
                  display: true,
                  text: 'Occupancy Rate (%)'
                }
              }
            }
          }
        });
      }

      // Initialize vehicle flow chart
      const flowCtx = document.getElementById('vehicleFlowChart');
      if (flowCtx) {
        new Chart(flowCtx, {
          type: 'bar',
          data: {
            labels: vehicleFlowData.map(item => new Date(item.timestamp).toLocaleTimeString()),
            datasets: [
              {
                label: 'Entries',
                data: vehicleFlowData.map(item => item.entries),
                backgroundColor: 'rgba(54, 162, 235, 0.6)',
                borderColor: 'rgba(54, 162, 235, 1)',
                borderWidth: 1
              },
              {
                label: 'Exits',
                data: vehicleFlowData.map(item => item.exits),
                backgroundColor: 'rgba(255, 99, 132, 0.6)',
                borderColor: 'rgba(255, 99, 132, 1)',
                borderWidth: 1
              }
            ]
          },
          options: {
            responsive: true,
            scales: {
              y: {
                beginAtZero: true,
                title: {
                  display: true,
                  text: 'Vehicle Count'
                }
              }
            }
          }
        });
      }

      // Initialize peak hours chart
      const peakCtx = document.getElementById('peakHoursChart');
      if (peakCtx) {
        new Chart(peakCtx, {
          type: 'bar',
          data: {
            labels: peakHoursData.map(item => `${item.startHour}:00-${item.endHour}:00`),
            datasets: [
              {
                label: 'Expected Vehicles',
                data: peakHoursData.map(item => item.expectedVehicles),
                backgroundColor: 'rgba(75, 192, 192, 0.6)',
                borderColor: 'rgba(75, 192, 192, 1)',
                borderWidth: 1
              },
              {
                label: 'Probability (%)',
                data: peakHoursData.map(item => item.probability * 100),
                backgroundColor: 'rgba(153, 102, 255, 0.6)',
                borderColor: 'rgba(153, 102, 255, 1)',
                borderWidth: 1,
                yAxisID: 'y1'
              }
            ]
          },
          options: {
            responsive: true,
            scales: {
              y: {
                beginAtZero: true,
                title: {
                  display: true,
                  text: 'Expected Vehicles'
                }
              },
              y1: {
                beginAtZero: true,
                max: 100,
                position: 'right',
                title: {
                  display: true,
                  text: 'Probability (%)'
                },
                grid: {
                  drawOnChartArea: false
                }
              }
            }
          }
        });
      }
    }
  }, [loading, revenueData, occupancyData, vehicleFlowData, peakHoursData]);

  if (loading) {
    return <div className="loading">Loading dashboard data...</div>;
  }

  return (
    <div className="analytics-dashboard">
      <h1>数据分析仪表盘</h1>
      
      <div className="dashboard-grid">
        {/* Revenue Trend */}
        <div className="dashboard-card">
          <h2>收入趋势</h2>
          <div className="chart-container">
            <canvas id="revenueChart"></canvas>
          </div>
        </div>

        {/* Occupancy Rate */}
        <div className="dashboard-card">
          <h2>占用率变化</h2>
          <div className="chart-container">
            <canvas id="occupancyChart"></canvas>
          </div>
        </div>

        {/* Vehicle Flow */}
        <div className="dashboard-card">
          <h2>车辆流量</h2>
          <div className="chart-container">
            <canvas id="vehicleFlowChart"></canvas>
          </div>
        </div>

        {/* Peak Hours Prediction */}
        <div className="dashboard-card">
          <h2>高峰时段预测</h2>
          <div className="chart-container">
            <canvas id="peakHoursChart"></canvas>
          </div>
        </div>
      </div>
    </div>
  );
};

export default AnalyticsDashboard;