// 路径规划算法

import { parkingData } from '../data/parkingData';

// 构建图结构
const buildGraph = (floorId) => {
  const floor = parkingData.floors.find(f => f.id === floorId);
  if (!floor) return { nodes: [], edges: [] };

  const nodes = [];
  const edges = [];
  const nodeIdMap = new Map();

  // 添加入口节点
  floor.entrances.forEach(entrance => {
    const nodeId = `entrance-${entrance.id}`;
    nodeIdMap.set(`${entrance.x},${entrance.y}`, nodeId);
    nodes.push({
      id: nodeId,
      x: entrance.x,
      y: entrance.y,
      type: 'entrance'
    });
  });

  // 添加出口节点
  floor.exits.forEach(exit => {
    const nodeId = `exit-${exit.id}`;
    nodeIdMap.set(`${exit.x},${exit.y}`, nodeId);
    nodes.push({
      id: nodeId,
      x: exit.x,
      y: exit.y,
      type: 'exit'
    });
  });

  // 添加停车位节点
  floor.sections.forEach(section => {
    section.parkingSpaces.forEach(space => {
      const nodeId = `space-${space.id}`;
      nodeIdMap.set(`${space.x},${space.y}`, nodeId);
      nodes.push({
        id: nodeId,
        x: space.x,
        y: space.y,
        type: 'parking',
        spaceId: space.id
      });
    });
  });

  // 添加通道节点和边
  floor.sections.forEach(section => {
    section.pathways.forEach(pathway => {
      for (let i = 0; i < pathway.points.length - 1; i++) {
        const [x1, y1] = pathway.points[i];
        const [x2, y2] = pathway.points[i + 1];
        
        // 计算距离
        const distance = Math.sqrt(Math.pow(x2 - x1, 2) + Math.pow(y2 - y1, 2));
        
        // 生成节点 ID
        const nodeId1 = nodeIdMap.get(`${x1},${y1}`) || `path-${x1}-${y1}`;
        const nodeId2 = nodeIdMap.get(`${x2},${y2}`) || `path-${x2}-${y2}`;
        
        // 添加节点（如果不存在）
        if (!nodeIdMap.has(`${x1},${y1}`)) {
          nodeIdMap.set(`${x1},${y1}`, nodeId1);
          nodes.push({
            id: nodeId1,
            x: x1,
            y: y1,
            type: 'path'
          });
        }
        
        if (!nodeIdMap.has(`${x2},${y2}`)) {
          nodeIdMap.set(`${x2},${y2}`, nodeId2);
          nodes.push({
            id: nodeId2,
            x: x2,
            y: y2,
            type: 'path'
          });
        }
        
        // 添加边
        edges.push({
          from: nodeId1,
          to: nodeId2,
          weight: distance
        });
        
        // 添加反向边
        edges.push({
          from: nodeId2,
          to: nodeId1,
          weight: distance
        });
      }
    });
  });

  return { nodes, edges };
};

// Dijkstra 算法实现
const dijkstra = (graph, startNodeId, endNodeId) => {
  const { nodes, edges } = graph;
  
  // 初始化距离和前驱节点
  const distances = {};
  const previous = {};
  const priorityQueue = [];
  
  // 初始化距离为无穷大
  nodes.forEach(node => {
    distances[node.id] = Infinity;
    previous[node.id] = null;
  });
  
  // 起点距离为 0
  distances[startNodeId] = 0;
  priorityQueue.push({ id: startNodeId, distance: 0 });
  
  while (priorityQueue.length > 0) {
    // 按距离排序，取最小距离的节点
    priorityQueue.sort((a, b) => a.distance - b.distance);
    const current = priorityQueue.shift();
    
    if (current.id === endNodeId) {
      // 找到终点，构建路径
      const path = [];
      let currentNode = endNodeId;
      while (currentNode) {
        path.unshift(currentNode);
        currentNode = previous[currentNode];
      }
      return path;
    }
    
    // 遍历所有边
    edges.forEach(edge => {
      if (edge.from === current.id) {
        const newDistance = distances[current.id] + edge.weight;
        if (newDistance < distances[edge.to]) {
          distances[edge.to] = newDistance;
          previous[edge.to] = current.id;
          
          // 更新优先队列
          const existingIndex = priorityQueue.findIndex(item => item.id === edge.to);
          if (existingIndex > -1) {
            priorityQueue[existingIndex].distance = newDistance;
          } else {
            priorityQueue.push({ id: edge.to, distance: newDistance });
          }
        }
      }
    });
  }
  
  // 没有找到路径
  return [];
};

// 获取节点坐标
const getNodeCoordinates = (graph, nodeId) => {
  const node = graph.nodes.find(n => n.id === nodeId);
  return node ? { x: node.x, y: node.y } : null;
};

// 生成路径坐标点
const generatePathCoordinates = (graph, path) => {
  return path.map(nodeId => {
    const coords = getNodeCoordinates(graph, nodeId);
    return coords ? [coords.x, coords.y] : null;
  }).filter(Boolean);
};

// 路径规划函数
export const findPath = (floorId, startType, startId, endType, endId) => {
  const graph = buildGraph(floorId);
  
  // 构建起点和终点的节点 ID
  let startNodeId, endNodeId;
  
  if (startType === 'entrance') {
    startNodeId = `entrance-${startId}`;
  } else if (startType === 'exit') {
    startNodeId = `exit-${startId}`;
  } else if (startType === 'parking') {
    startNodeId = `space-${startId}`;
  }
  
  if (endType === 'entrance') {
    endNodeId = `entrance-${endId}`;
  } else if (endType === 'exit') {
    endNodeId = `exit-${endId}`;
  } else if (endType === 'parking') {
    endNodeId = `space-${endId}`;
  }
  
  // 执行路径规划
  const path = dijkstra(graph, startNodeId, endNodeId);
  const pathCoordinates = generatePathCoordinates(graph, path);
  
  return {
    path,
    pathCoordinates,
    distance: calculatePathDistance(pathCoordinates)
  };
};

// 计算路径距离
const calculatePathDistance = (coordinates) => {
  let distance = 0;
  for (let i = 0; i < coordinates.length - 1; i++) {
    const [x1, y1] = coordinates[i];
    const [x2, y2] = coordinates[i + 1];
    distance += Math.sqrt(Math.pow(x2 - x1, 2) + Math.pow(y2 - y1, 2));
  }
  return distance;
};

// 找到最近的可用停车位
export const findNearestAvailableParking = (floorId, entranceId) => {
  const floor = parkingData.floors.find(f => f.id === floorId);
  if (!floor) return null;
  
  // 获取所有可用停车位
  const availableSpaces = [];
  floor.sections.forEach(section => {
    section.parkingSpaces.forEach(space => {
      if (space.status === 'available') {
        availableSpaces.push(space);
      }
    });
  });
  
  if (availableSpaces.length === 0) return null;
  
  // 找到最近的停车位
  let nearestSpace = null;
  let minDistance = Infinity;
  
  availableSpaces.forEach(space => {
    const pathResult = findPath(floorId, 'entrance', entranceId, 'parking', space.id);
    if (pathResult.distance < minDistance) {
      minDistance = pathResult.distance;
      nearestSpace = {
        ...space,
        distance: minDistance,
        path: pathResult
      };
    }
  });
  
  return nearestSpace;
};
