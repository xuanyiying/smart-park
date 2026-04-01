import React, { useState, useEffect } from 'react';
import { MapContainer, TileLayer, SVGOverlay, CircleMarker, Popup, Polyline } from 'react-leaflet';
import { parkingData, getParkingFloors, getParkingSpacesByFloor } from '../data/parkingData';
import { findPath, findNearestAvailableParking } from '../utils/pathfinding';
import { positioningSystem } from '../utils/positioning';
import RealTimeNavigation from './RealTimeNavigation';

const IndoorMap = () => {
  const [selectedFloor, setSelectedFloor] = useState('floor-1');
  const [path, setPath] = useState([]);
  const [nearestParking, setNearestParking] = useState(null);
  const [isNavigating, setIsNavigating] = useState(false);
  const [navigationParams, setNavigationParams] = useState(null);
  const [userPosition, setUserPosition] = useState(null);
  const [nearbyParkingSpace, setNearbyParkingSpace] = useState(null);
  const floors = getParkingFloors();
  const parkingSpaces = getParkingSpacesByFloor(selectedFloor);
  const currentFloor = parkingData.floors.find(f => f.id === selectedFloor);

  // 初始化定位系统
  useEffect(() => {
    // 开始位置追踪
    positioningSystem.startTracking((position) => {
      setUserPosition(position);
      
      // 检查是否靠近停车位
      const nearbySpace = positioningSystem.checkIfNearParkingSpace(position);
      setNearbyParkingSpace(nearbySpace);
    });

    // 切换楼层时更新定位系统
    positioningSystem.setFloor(selectedFloor);

    // 清理函数
    return () => {
      positioningSystem.stopTracking();
    };
  }, [selectedFloor]);

  // 寻找最近的可用停车位
  const handleFindNearestParking = () => {
    if (!currentFloor || currentFloor.entrances.length === 0) return;
    
    const entranceId = currentFloor.entrances[0].id;
    const nearest = findNearestAvailableParking(selectedFloor, entranceId);
    
    if (nearest) {
      setNearestParking(nearest);
      setPath(nearest.path.pathCoordinates);
      setNavigationParams({
        floorId: selectedFloor,
        startType: 'entrance',
        startId: entranceId,
        endType: 'parking',
        endId: nearest.id
      });
    }
  };

  // 清除路径
  const handleClearPath = () => {
    setPath([]);
    setNearestParking(null);
    setIsNavigating(false);
    setNavigationParams(null);
  };

  // 开始实时导航
  const startRealTimeNavigation = () => {
    if (nearestParking && navigationParams) {
      setIsNavigating(true);
    }
  };

  // 导航结束回调
  const handleNavigationEnd = () => {
    setIsNavigating(false);
  };

  // 生成 SVG 室内地图
  const generateIndoorMapSVG = () => {
    if (!currentFloor) return '';

    const { width, height, sections, pathways, exits, entrances } = currentFloor;
    const scale = 2; // 缩放比例

    return (
      <svg width={width * scale} height={height * scale} viewBox={`0 0 ${width} ${height}`}>
        {/* 绘制区域 */}
        {sections.map(section => (
          <g key={section.id}>
            <rect
              x={section.startX}
              y={section.startY}
              width={section.endX - section.startX}
              height={section.endY - section.startY}
              fill="#f0f0f0"
              stroke="#ccc"
              strokeWidth="0.5"
            />
            <text
              x={section.startX + 10}
              y={section.startY + 15}
              fontSize="12"
              fill="#333"
            >
              {section.name}
            </text>
          </g>
        ))}

        {/* 绘制通道 */}
        {pathways.map(pathway => (
          <polyline
            key={pathway.id}
            points={pathway.points.map(p => `${p[0]},${p[1]}`).join(' ')}
            fill="none"
            stroke="#ddd"
            strokeWidth="8"
          />
        ))}

        {/* 绘制出口 */}
        {exits.map(exit => (
          <circle
            key={exit.id}
            cx={exit.x}
            cy={exit.y}
            r="3"
            fill="#4CAF50"
          />
        ))}

        {/* 绘制入口 */}
        {entrances.map(entrance => (
          <circle
            key={entrance.id}
            cx={entrance.x}
            cy={entrance.y}
            r="3"
            fill="#2196F3"
          />
        ))}

        {/* 绘制停车位 */}
        {parkingSpaces.map(space => (
          <rect
            key={space.id}
            x={space.x - 2}
            y={space.y - 2}
            width="4"
            height="4"
            fill={space.status === 'available' ? '#4CAF50' : '#F44336'}
            stroke="#333"
            strokeWidth="0.5"
          />
        ))}
      </svg>
    );
  };

  // 将停车场坐标转换为地理坐标
  const getGeoPosition = (x, y) => {
    if (!currentFloor) return [0, 0];
    // 简单的坐标转换，实际应用中可能需要更复杂的转换
    const baseLat = parkingData.location[0];
    const baseLng = parkingData.location[1];
    const latOffset = (y - currentFloor.height / 2) * 0.0001;
    const lngOffset = (x - currentFloor.width / 2) * 0.0001;
    return [baseLat + latOffset, baseLng + lngOffset];
  };

  // 转换路径坐标为地理坐标
  const pathGeoCoordinates = path.map(([x, y]) => getGeoPosition(x, y));

  return (
    <div className="indoor-map-container">
      <div className="floor-selector">
        {floors.map(floor => (
          <button
            key={floor.id}
            className={selectedFloor === floor.id ? 'active' : ''}
            onClick={() => {
              setSelectedFloor(floor.id);
              setPath([]);
              setNearestParking(null);
              setIsNavigating(false);
              setNavigationParams(null);
            }}
          >
            {floor.name}
          </button>
        ))}
        <button onClick={handleFindNearestParking} className="control-button">
          查找最近停车位
        </button>
        {nearestParking && !isNavigating && (
          <button onClick={startRealTimeNavigation} className="control-button">
            开始导航
          </button>
        )}
        <button onClick={handleClearPath} className="control-button">
          清除路径
        </button>
      </div>
      <MapContainer
        center={parkingData.location}
        zoom={18}
        style={{ height: '80vh', width: '100%' }}
      >
        <TileLayer
          attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
          url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
        />
        
        {/* 室内地图 SVG 覆盖层 */}
        {currentFloor && (
          <SVGOverlay
            bounds={[
              getGeoPosition(0, currentFloor.height),
              getGeoPosition(currentFloor.width, 0)
            ]}
            data={generateIndoorMapSVG()}
          />
        )}

        {/* 路径显示 */}
        {pathGeoCoordinates.length > 1 && (
          <Polyline
            positions={pathGeoCoordinates}
            color="#FF5722"
            weight={4}
            opacity={0.8}
          />
        )}

        {/* 停车位标记 */}
        {parkingSpaces.map(space => (
          <CircleMarker
            key={space.id}
            center={getGeoPosition(space.x, space.y)}
            radius={5}
            fillColor={space.status === 'available' ? '#4CAF50' : '#F44336'}
            color="#333"
            weight={1}
            fillOpacity={0.8}
          >
            <Popup>
              <div>
                <strong>停车位: {space.id}</strong>
                <p>状态: {space.status === 'available' ? '可用' : '已占用'}</p>
                <p>区域: {space.section}</p>
                <p>楼层: {space.floor}</p>
              </div>
            </Popup>
          </CircleMarker>
        ))}

        {/* 最近停车位标记 */}
        {nearestParking && (
          <CircleMarker
            center={getGeoPosition(nearestParking.x, nearestParking.y)}
            radius={8}
            fillColor="#FFC107"
            color="#333"
            weight={2}
            fillOpacity={0.9}
          >
            <Popup>
              <div>
                <strong>最近可用停车位: {nearestParking.id}</strong>
                <p>距离: {nearestParking.distance.toFixed(1)} 米</p>
                <p>区域: {nearestParking.section}</p>
                <p>楼层: {nearestParking.floor}</p>
              </div>
            </Popup>
          </CircleMarker>
        )}

        {/* 出口标记 */}
        {currentFloor?.exits.map(exit => (
          <CircleMarker
            key={exit.id}
            center={getGeoPosition(exit.x, exit.y)}
            radius={6}
            fillColor="#4CAF50"
            color="#333"
            weight={1}
            fillOpacity={0.8}
          >
            <Popup>
              <div>
                <strong>{exit.name}</strong>
              </div>
            </Popup>
          </CircleMarker>
        ))}

        {/* 入口标记 */}
        {currentFloor?.entrances.map(entrance => (
          <CircleMarker
            key={entrance.id}
            center={getGeoPosition(entrance.x, entrance.y)}
            radius={6}
            fillColor="#2196F3"
            color="#333"
            weight={1}
            fillOpacity={0.8}
          >
            <Popup>
              <div>
                <strong>{entrance.name}</strong>
              </div>
            </Popup>
          </CircleMarker>
        ))}

        {/* 用户位置标记 */}
        {userPosition && !isNavigating && (
          <CircleMarker
            center={getGeoPosition(userPosition.x, userPosition.y)}
            radius={8}
            fillColor="#2196F3"
            color="#fff"
            weight={2}
            fillOpacity={0.9}
          >
            <Popup>
              <div>
                <strong>当前位置</strong>
                <p>坐标: ({userPosition.x.toFixed(1)}, {userPosition.y.toFixed(1)})</p>
                <p>楼层: {userPosition.floor}</p>
                <p>精度: {userPosition.accuracy.toFixed(1)} 米</p>
                {nearbyParkingSpace && (
                  <div style={{ marginTop: '8px' }}>
                    <strong>附近停车位: {nearbyParkingSpace.id}</strong>
                    <p>距离: {nearbyParkingSpace.distance.toFixed(1)} 米</p>
                    <p>状态: {nearbyParkingSpace.status === 'available' ? '可用' : '已占用'}</p>
                  </div>
                )}
              </div>
            </Popup>
          </CircleMarker>
        )}

        {/* 实时导航 */}
        {isNavigating && navigationParams && (
          <RealTimeNavigation
            floorId={navigationParams.floorId}
            startType={navigationParams.startType}
            startId={navigationParams.startId}
            endType={navigationParams.endType}
            endId={navigationParams.endId}
            getGeoPosition={getGeoPosition}
            onNavigationEnd={handleNavigationEnd}
          />
        )}
      </MapContainer>
    </div>
  );
};

export default IndoorMap;