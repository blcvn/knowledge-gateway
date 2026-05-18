import type { SubgraphData } from '../types/graph';

export const graphMock = {
  subgraph: {
    nodes: [{ id: 'n1', label: 'Person', type: 'Entity' }],
    edges: [{ id: 'e1', source: 'n1', target: 'n1', type: 'KNOWS' }]
  } as SubgraphData,
  timeline: [],
};
