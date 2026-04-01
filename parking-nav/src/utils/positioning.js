// 车位级定位系统

import { parkingData } from '../data/parkingData';

// 模拟定位数据
const mockPositioningData = {
  // 模拟蓝牙信标
  beacons: [
    { id: 'beacon-001', x: 10, y: 10, floor: 'floor-1', rssi: -60 },
    { id: 'beacon-002', x: 40, y: 10, floor: 'floor-1', rssi: -65 },
    { id: 'beacon-003', x: 10, y: 40, floor: 'floor-1', rssi: -70 },
    { id: 'beacon-004', x: 40, y: 40, floor: 'floor-1', rssi: -75 },
    { id: 'beacon-005', x: 10, y: 10, floor: 'floor-2', rssi: -60 },
    { id: 'beacon-006', x: 40, y: 10, floor: 'floor-2', rssi: -65 },
  ],
  
  // 模拟WiFi接入点
  wifiAccessPoints: [
    { bssid: '00:11:22:33:44:55', ssid: 'Parking-WiFi-1', rssi: -55, floor: 'floor-1' },
    { bssid: '66:77:88:99:AA:BB', ssid: 'Parking-WiFi-2', rssi: -60, floor: 'floor-1' },
    { bssid: 'CC:DD:EE:FF:00:11', ssid: 'Parking-WiFi-3', rssi: -65, floor: 'floor-2' },
  ],
};

// 模拟用户位置更新
class PositioningSystem {
  constructor() {
    this.currentPosition = null;
    this.currentFloor = 'floor-1';
    this.positionUpdateCallback = null;
    this.isTracking = false;
    this.updateInterval = null;
  }

  // 开始位置追踪
  startTracking(callback) {
    this.positionUpdateCallback = callback;
    this.isTracking = true;
    
    // 模拟位置更新，每2秒更新一次
    this.updateInterval = setInterval(() => {
      this.updatePosition();
    }, 2000);
  }

  // 停止位置追踪
  stopTracking() {
    this.isTracking = false;
    if (this.updateInterval) {
      clearInterval(this.updateInterval);
    }
  }

  // 更新位置
  updatePosition() {
    if (!this.isTracking) return;
    
    // 基于蓝牙信标和WiFi信号强度进行三角定位
    const position = this.calculatePosition();
    this.currentPosition = position;
    
    if (this.positionUpdateCallback) {
      this.positionUpdateCallback(position);
    }
  }

  // 计算位置（模拟三角定位）
  calculatePosition() {
    // 获取当前楼层的信标
    const floorBeacons = mockPositioningData.beacons.filter(
      beacon => beacon.floor === this.currentFloor
    );
    
    // 模拟基于信标的定位
    if (floorBeacons.length > 0) {
      // 简单的加权平均算法
      let weightedX = 0;
      let weightedY = 0;
      let totalWeight = 0;
      
      floorBeacons.forEach(beacon => {
        // RSSI值越接近0，信号越强，权重越大
        const weight = 1 / (Math.abs(beacon.rssi) / 100);
        weightedX += beacon.x * weight;
        weightedY += beacon.y * weight;
        totalWeight += weight;
      });
      
      const x = weightedX / totalWeight;
      const y = weightedY / totalWeight;
      
      // 添加一些随机误差模拟真实定位
      const error = 0.5; // 误差范围
      const finalX = x + (Math.random() * error * 2 - error);
      const finalY = y + (Math.random() * error * 2 - error);
      
      return {
        x: Math.max(0, Math.min(100, finalX)), // 限制在停车场范围内
        y: Math.max(0, Math.min(80, finalY)),
        floor: this.currentFloor,
        accuracy: error * 2, // 定位精度
        timestamp: new Date().getTime()
      };
    }
    
    // 默认位置
    return {
      x: 25,
      y: 5,
      floor: this.currentFloor,
      accuracy: 2,
      timestamp: new Date().getTime()
    };
  }

  // 切换楼层
  setFloor(floorId) {
    this.currentFloor = floorId;
    this.updatePosition();
  }

  // 获取当前位置
  getCurrentPosition() {
    return this.currentPosition;
  }

  // 检查用户是否在停车位附近
  checkIfNearParkingSpace(position) {
    const floor = parkingData.floors.find(f => f.id === position.floor);
    if (!floor) return null;
    
    let nearestSpace = null;
    let minDistance = Infinity;
    
    floor.sections.forEach(section => {
      section.parkingSpaces.forEach(space => {
        const distance = Math.sqrt(
          Math.pow(space.x - position.x, 2) + Math.pow(space.y - position.y, 2)
        );
        
        if (distance < minDistance && distance < 3) { // 3米范围内
          minDistance = distance;
          nearestSpace = {
            ...space,
            distance
          };
        }
      });
    });
    
    return nearestSpace;
  }

  // 更新停车位状态
  updateParkingSpaceStatus(spaceId, status) {
    parkingData.floors.forEach(floor => {
      floor.sections.forEach(section => {
        section.parkingSpaces.forEach(space => {
          if (space.id === spaceId) {
            space.status = status;
          }
        });
      });
    });
  }

  // 获取停车位状态
  getParkingSpaceStatus(spaceId) {
    for (const floor of parkingData.floors) {
      for (const section of floor.sections) {
        for (const space of section.parkingSpaces) {
          if (space.id === spaceId) {
            return space.status;
          }
        }
      }
    }
    return null;
  }
}

// 导出定位系统实例
export const positioningSystem = new PositioningSystem();

// 模拟停车位状态变化
export const simulateParkingSpaceStatusChanges = () => {
  // 随机改变一些停车位的状态
  parkingData.floors.forEach(floor => {
    floor.sections.forEach(section => {
      section.parkingSpaces.forEach(space => {
        // 10% 的概率改变状态
        if (Math.random() < 0.1) {
          space.status = space.status === 'available' ? 'occupied' : 'available';
        }
      });
    });
  });
};

// 每30秒模拟一次停车位状态变化
setInterval(simulateParkingSpaceStatusChanges, 30000);
