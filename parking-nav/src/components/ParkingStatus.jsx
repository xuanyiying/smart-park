import React from 'react';
import { parkingData } from '../data/parkingData';

const ParkingStatus = () => {
  // 计算停车位总数和可用数量
  const calculateParkingStats = () => {
    let totalSpaces = 0;
    let availableSpaces = 0;
    const floorStats = [];

    parkingData.floors.forEach(floor => {
      let floorTotal = 0;
      let floorAvailable = 0;

      floor.sections.forEach(section => {
        section.parkingSpaces.forEach(space => {
          floorTotal++;
          if (space.status === 'available') {
            floorAvailable++;
          }
        });
      });

      totalSpaces += floorTotal;
      availableSpaces += floorAvailable;

      floorStats.push({
        floorId: floor.id,
        floorName: floor.name,
        total: floorTotal,
        available: floorAvailable,
        occupied: floorTotal - floorAvailable,
        occupancyRate: ((floorTotal - floorAvailable) / floorTotal * 100).toFixed(1)
      });
    });

    return {
      totalSpaces,
      availableSpaces,
      occupiedSpaces: totalSpaces - availableSpaces,
      occupancyRate: ((totalSpaces - availableSpaces) / totalSpaces * 100).toFixed(1),
      floorStats
    };
  };

  const stats = calculateParkingStats();

  return (
    <div className="parking-status">
      <h2>停车场状态</h2>
      <div className="status-overview">
        <div className="status-card">
          <div className="status-value">{stats.totalSpaces}</div>
          <div className="status-label">总停车位</div>
        </div>
        <div className="status-card available">
          <div className="status-value">{stats.availableSpaces}</div>
          <div className="status-label">可用</div>
        </div>
        <div className="status-card occupied">
          <div className="status-value">{stats.occupiedSpaces}</div>
          <div className="status-label">已占用</div>
        </div>
        <div className="status-card rate">
          <div className="status-value">{stats.occupancyRate}%</div>
          <div className="status-label">占用率</div>
        </div>
      </div>
      <div className="floor-stats">
        {stats.floorStats.map(floor => (
          <div key={floor.floorId} className="floor-stat-card">
            <h3>{floor.floorName}</h3>
            <div className="floor-stat-details">
              <div className="floor-stat-item">
                <span className="floor-stat-label">总车位:</span>
                <span className="floor-stat-value">{floor.total}</span>
              </div>
              <div className="floor-stat-item">
                <span className="floor-stat-label">可用:</span>
                <span className="floor-stat-value available">{floor.available}</span>
              </div>
              <div className="floor-stat-item">
                <span className="floor-stat-label">已占用:</span>
                <span className="floor-stat-value occupied">{floor.occupied}</span>
              </div>
              <div className="floor-stat-item">
                <span className="floor-stat-label">占用率:</span>
                <span className="floor-stat-value rate">{floor.occupancyRate}%</span>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

export default ParkingStatus;