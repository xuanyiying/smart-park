// 停车场数据模型

export const parkingData = {
  id: 'parking-001',
  name: '中央停车场',
  location: [39.9042, 116.4074],
  floors: [
    {
      id: 'floor-1',
      name: 'B1 层',
      level: -1,
      width: 100, // 米
      height: 80, // 米
      sections: [
        {
          id: 'section-A',
          name: 'A 区',
          startX: 0,
          startY: 0,
          endX: 50,
          endY: 80,
          parkingSpaces: [
            { id: 'A-001', x: 5, y: 5, status: 'available' },
            { id: 'A-002', x: 15, y: 5, status: 'occupied' },
            { id: 'A-003', x: 25, y: 5, status: 'available' },
            { id: 'A-004', x: 35, y: 5, status: 'available' },
            { id: 'A-005', x: 45, y: 5, status: 'occupied' },
            { id: 'A-006', x: 5, y: 15, status: 'available' },
            { id: 'A-007', x: 15, y: 15, status: 'available' },
            { id: 'A-008', x: 25, y: 15, status: 'occupied' },
            { id: 'A-009', x: 35, y: 15, status: 'available' },
            { id: 'A-010', x: 45, y: 15, status: 'available' },
          ],
          pathways: [
            { id: 'path-A1', points: [[0, 0], [50, 0], [50, 80], [0, 80], [0, 0]] },
            { id: 'path-A2', points: [[25, 0], [25, 80]] },
          ],
        },
        {
          id: 'section-B',
          name: 'B 区',
          startX: 50,
          startY: 0,
          endX: 100,
          endY: 80,
          parkingSpaces: [
            { id: 'B-001', x: 55, y: 5, status: 'occupied' },
            { id: 'B-002', x: 65, y: 5, status: 'available' },
            { id: 'B-003', x: 75, y: 5, status: 'available' },
            { id: 'B-004', x: 85, y: 5, status: 'occupied' },
            { id: 'B-005', x: 95, y: 5, status: 'available' },
            { id: 'B-006', x: 55, y: 15, status: 'available' },
            { id: 'B-007', x: 65, y: 15, status: 'occupied' },
            { id: 'B-008', x: 75, y: 15, status: 'available' },
            { id: 'B-009', x: 85, y: 15, status: 'available' },
            { id: 'B-010', x: 95, y: 15, status: 'occupied' },
          ],
          pathways: [
            { id: 'path-B1', points: [[50, 0], [100, 0], [100, 80], [50, 80], [50, 0]] },
            { id: 'path-B2', points: [[75, 0], [75, 80]] },
          ],
        },
      ],
      exits: [
        { id: 'exit-1', x: 25, y: 80, name: '电梯口' },
        { id: 'exit-2', x: 75, y: 80, name: '楼梯口' },
      ],
      entrances: [
        { id: 'entrance-1', x: 25, y: 0, name: '入口' },
      ],
    },
    {
      id: 'floor-2',
      name: 'B2 层',
      level: -2,
      width: 100,
      height: 80,
      sections: [
        {
          id: 'section-C',
          name: 'C 区',
          startX: 0,
          startY: 0,
          endX: 50,
          endY: 80,
          parkingSpaces: [
            { id: 'C-001', x: 5, y: 5, status: 'available' },
            { id: 'C-002', x: 15, y: 5, status: 'available' },
            { id: 'C-003', x: 25, y: 5, status: 'occupied' },
            { id: 'C-004', x: 35, y: 5, status: 'available' },
            { id: 'C-005', x: 45, y: 5, status: 'available' },
          ],
          pathways: [
            { id: 'path-C1', points: [[0, 0], [50, 0], [50, 80], [0, 80], [0, 0]] },
          ],
        },
        {
          id: 'section-D',
          name: 'D 区',
          startX: 50,
          startY: 0,
          endX: 100,
          endY: 80,
          parkingSpaces: [
            { id: 'D-001', x: 55, y: 5, status: 'occupied' },
            { id: 'D-002', x: 65, y: 5, status: 'available' },
            { id: 'D-003', x: 75, y: 5, status: 'occupied' },
            { id: 'D-004', x: 85, y: 5, status: 'available' },
            { id: 'D-005', x: 95, y: 5, status: 'available' },
          ],
          pathways: [
            { id: 'path-D1', points: [[50, 0], [100, 0], [100, 80], [50, 80], [50, 0]] },
          ],
        },
      ],
      exits: [
        { id: 'exit-3', x: 25, y: 80, name: '电梯口' },
        { id: 'exit-4', x: 75, y: 80, name: '楼梯口' },
      ],
      entrances: [
        { id: 'entrance-2', x: 25, y: 0, name: '入口' },
      ],
    },
  ],
};

// 获取可用停车位
export const getAvailableParkingSpaces = (floorId) => {
  const floor = parkingData.floors.find(f => f.id === floorId);
  if (!floor) return [];
  
  let availableSpaces = [];
  floor.sections.forEach(section => {
    section.parkingSpaces.forEach(space => {
      if (space.status === 'available') {
        availableSpaces.push({
          ...space,
          section: section.name,
          floor: floor.name,
        });
      }
    });
  });
  return availableSpaces;
};

// 获取停车场楼层
export const getParkingFloors = () => {
  return parkingData.floors.map(floor => ({
    id: floor.id,
    name: floor.name,
    level: floor.level,
  }));
};

// 获取指定楼层的所有停车位
export const getParkingSpacesByFloor = (floorId) => {
  const floor = parkingData.floors.find(f => f.id === floorId);
  if (!floor) return [];
  
  let spaces = [];
  floor.sections.forEach(section => {
    section.parkingSpaces.forEach(space => {
      spaces.push({
        ...space,
        section: section.name,
        floor: floor.name,
      });
    });
  });
  return spaces;
};
