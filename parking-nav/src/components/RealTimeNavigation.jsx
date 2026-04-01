import React, { useState, useEffect } from 'react';
import { CircleMarker, Polyline, Popup } from 'react-leaflet';
import { findPath } from '../utils/pathfinding';

const RealTimeNavigation = ({ floorId, startType, startId, endType, endId, getGeoPosition, onNavigationEnd }) => {
  const [userPosition, setUserPosition] = useState(null);
  const [path, setPath] = useState([]);
  const [pathGeoCoordinates, setPathGeoCoordinates] = useState([]);
  const [currentIndex, setCurrentIndex] = useState(0);
  const [navigationInstructions, setNavigationInstructions] = useState([]);
  const [isNavigating, setIsNavigating] = useState(false);

  // 初始化导航
  useEffect(() => {
    if (floorId && startType && startId && endType && endId) {
      const pathResult = findPath(floorId, startType, startId, endType, endId);
      setPath(pathResult.pathCoordinates);
      
      const geoPath = pathResult.pathCoordinates.map(([x, y]) => getGeoPosition(x, y));
      setPathGeoCoordinates(geoPath);
      
      if (pathResult.pathCoordinates.length > 0) {
        setUserPosition(pathResult.pathCoordinates[0]);
        setCurrentIndex(0);
        generateNavigationInstructions(pathResult.pathCoordinates);
      }
    }
  }, [floorId, startType, startId, endType, endId, getGeoPosition]);

  // 生成导航指令
  const generateNavigationInstructions = (pathCoordinates) => {
    const instructions = [];
    
    for (let i = 0; i < pathCoordinates.length - 2; i++) {
      const [x1, y1] = pathCoordinates[i];
      const [x2, y2] = pathCoordinates[i + 1];
      const [x3, y3] = pathCoordinates[i + 2];
      
      // 计算方向变化
      const dx1 = x2 - x1;
      const dy1 = y2 - y1;
      const dx2 = x3 - x2;
      const dy2 = y3 - y2;
      
      // 计算角度变化
      const angle1 = Math.atan2(dy1, dx1) * 180 / Math.PI;
      const angle2 = Math.atan2(dy2, dx2) * 180 / Math.PI;
      const angleDiff = (angle2 - angle1 + 360) % 360;
      
      let instruction = '';
      if (angleDiff > 45 && angleDiff < 135) {
        instruction = '右转';
      } else if (angleDiff > 135 && angleDiff < 225) {
        instruction = '直行';
      } else if (angleDiff > 225 && angleDiff < 315) {
        instruction = '左转';
      } else {
        instruction = '直行';
      }
      
      instructions.push({
        point: [x2, y2],
        instruction,
        distance: Math.sqrt(Math.pow(dx1, 2) + Math.pow(dy1, 2))
      });
    }
    
    // 添加到达终点的指令
    if (pathCoordinates.length > 1) {
      instructions.push({
        point: pathCoordinates[pathCoordinates.length - 1],
        instruction: '到达目的地',
        distance: 0
      });
    }
    
    setNavigationInstructions(instructions);
  };

  // 开始导航
  const startNavigation = () => {
    setIsNavigating(true);
  };

  // 停止导航
  const stopNavigation = () => {
    setIsNavigating(false);
  };

  // 模拟用户移动
  useEffect(() => {
    let interval;
    if (isNavigating && path.length > currentIndex + 1) {
      interval = setInterval(() => {
        setCurrentIndex(prevIndex => {
          const newIndex = prevIndex + 1;
          if (newIndex < path.length) {
            setUserPosition(path[newIndex]);
            return newIndex;
          } else {
            setIsNavigating(false);
            if (onNavigationEnd) {
              onNavigationEnd();
            }
            return prevIndex;
          }
        });
      }, 2000); // 每2秒移动一次
    }
    
    return () => clearInterval(interval);
  }, [isNavigating, path, currentIndex, onNavigationEnd]);

  // 获取当前导航指令
  const currentInstruction = navigationInstructions.find(instr => {
    if (!userPosition) return false;
    const [x, y] = instr.point;
    return Math.abs(x - userPosition[0]) < 1 && Math.abs(y - userPosition[1]) < 1;
  });

  if (!path.length) {
    return null;
  }

  return (
    <>
      {/* 导航路径 */}
      {pathGeoCoordinates.length > 1 && (
        <Polyline
          positions={pathGeoCoordinates}
          color="#FF5722"
          weight={4}
          opacity={0.8}
        />
      )}

      {/* 用户位置标记 */}
      {userPosition && (
        <CircleMarker
          center={getGeoPosition(userPosition[0], userPosition[1])}
          radius={8}
          fillColor="#2196F3"
          color="#fff"
          weight={2}
          fillOpacity={0.9}
        >
          <Popup>
            <div>
              <strong>当前位置</strong>
              {currentInstruction && (
                <div style={{ marginTop: '8px' }}>
                  <p>导航指令: {currentInstruction.instruction}</p>
                  {currentInstruction.distance > 0 && (
                    <p>距离: {currentInstruction.distance.toFixed(1)} 米</p>
                  )}
                </div>
              )}
              <div style={{ marginTop: '12px', display: 'flex', gap: '8px' }}>
                <button 
                  onClick={startNavigation} 
                  disabled={isNavigating}
                  style={{
                    padding: '6px 12px',
                    backgroundColor: '#4CAF50',
                    color: 'white',
                    border: 'none',
                    borderRadius: '4px',
                    cursor: isNavigating ? 'not-allowed' : 'pointer'
                  }}
                >
                  开始导航
                </button>
                <button 
                  onClick={stopNavigation} 
                  disabled={!isNavigating}
                  style={{
                    padding: '6px 12px',
                    backgroundColor: '#F44336',
                    color: 'white',
                    border: 'none',
                    borderRadius: '4px',
                    cursor: !isNavigating ? 'not-allowed' : 'pointer'
                  }}
                >
                  停止导航
                </button>
              </div>
            </div>
          </Popup>
        </CircleMarker>
      )}

      {/* 导航指令面板 */}
      {isNavigating && currentInstruction && (
        <div style={{
          position: 'absolute',
          bottom: '20px',
          left: '50%',
          transform: 'translateX(-50%)',
          backgroundColor: 'rgba(255, 255, 255, 0.9)',
          padding: '12px',
          borderRadius: '8px',
          boxShadow: '0 2px 10px rgba(0, 0, 0, 0.1)',
          zIndex: 1000,
          minWidth: '200px',
          textAlign: 'center'
        }}>
          <h3 style={{ margin: '0 0 8px 0', color: '#333' }}>导航指引</h3>
          <p style={{ margin: '0', fontSize: '16px', fontWeight: 'bold' }}>{currentInstruction.instruction}</p>
          {currentInstruction.distance > 0 && (
            <p style={{ margin: '4px 0 0 0', fontSize: '14px', color: '#666' }}>
              距离: {currentInstruction.distance.toFixed(1)} 米
            </p>
          )}
        </div>
      )}
    </>
  );
};

export default RealTimeNavigation;